package pirepvaluemappings

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"time"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

type liveryCatalog interface {
	ListAll(ctx context.Context) ([]gameliveries.AircraftLivery, error)
	ListForCatalog(ctx context.Context, kind, search string, limit, offset int) ([]gameliveries.AircraftLivery, int, error)
	WriteCatalogCSV(ctx context.Context, w io.Writer) error
}


type Service struct {
	repo    *Repository
	liveries liveryCatalog
	lock    *vaconfig.Repository
}

func NewService(repo *Repository, liveries liveryCatalog, lock *vaconfig.Repository) *Service {
	return &Service{repo: repo, liveries: liveries, lock: lock}
}

type View struct {
	Revision  int64     `json:"revision"`
	Document  Document  `json:"document"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	Configured bool     `json:"configured"`
}

func (s *Service) Get(ctx context.Context, vaID string) (View, error) {
	row, err := s.repo.Get(ctx, vaID)
	if errors.Is(err, ErrNotFound) {
		return View{Revision: 0, Document: EmptyDocument(), Configured: false}, nil
	}
	if err != nil {
		return View{}, err
	}
	doc := DocumentFromRow(*row)
	return View{
		Revision:   row.Revision,
		Document:   doc,
		UpdatedAt:  row.UpdatedAt,
		UpdatedBy:  row.UpdatedBy,
		Configured: len(row.AircraftMappings) > 0 || len(row.AirlineMappings) > 0 || row.DefaultAircraft != nil || row.DefaultAirline != nil,
	}, nil
}

type PreviewInput struct {
	Document    *Document    `json:"document,omitempty"`
	AircraftID  string       `json:"aircraftId,omitempty"`
	LiveryID    string       `json:"liveryId,omitempty"`
}

type PreviewResult struct {
	Revision       int64         `json:"revision"`
	Normalized     Document      `json:"normalized"`
	Diff           Diff          `json:"diff"`
	Sample         *ResolveResult `json:"sample,omitempty"`
}

func (s *Service) Preview(ctx context.Context, vaID string, in PreviewInput) (PreviewResult, error) {
	current, err := s.Get(ctx, vaID)
	if err != nil {
		return PreviewResult{}, err
	}
	proposed := current.Document
	if in.Document != nil {
		proposed = in.Document.Normalize()
		if err := ValidateDocument(proposed); err != nil {
			return PreviewResult{}, err
		}
	}
	rows, err := s.liveries.ListAll(ctx)
	if err != nil {
		return PreviewResult{}, err
	}
	catalog := BuildCatalogNames(rows)
	out := PreviewResult{
		Revision:   current.Revision,
		Normalized: proposed,
		Diff:       DiffDocuments(current.Document, proposed, catalog),
	}
	if in.AircraftID != "" || in.LiveryID != "" {
		sample := s.resolveIDs(rows, ConfigFromDocument(proposed), ResolveInput{
			AircraftID: in.AircraftID,
			LiveryID:   in.LiveryID,
		})
		out.Sample = &sample
	}
	return out, nil
}

func ConfigFromDocument(doc Document) Config {
	row, err := RowFromDocument(doc.Normalize())
	if err != nil {
		return Config{AircraftMappings: map[string]string{}, AirlineMappings: map[string]string{}}
	}
	return ConfigFromRow(&row)
}

func (s *Service) resolveIDs(rows []gameliveries.AircraftLivery, cfg Config, input ResolveInput) ResolveResult {
	var livery gameliveries.AircraftLivery
	for _, row := range rows {
		if input.LiveryID != "" && row.LiveryID == input.LiveryID {
			livery = row
			break
		}
	}
	if livery.LiveryID == "" && input.AircraftID != "" {
		for _, row := range rows {
			if row.AircraftID == input.AircraftID {
				livery = row
				break
			}
		}
	}
	return Resolve(cfg, livery, input)
}

func (s *Service) Save(ctx context.Context, vaID, actor string, expectedRevision int64, doc Document) (View, error) {
	doc = doc.Normalize()
	if err := ValidateDocument(doc); err != nil {
		return View{}, err
	}
	payload, err := RowFromDocument(doc)
	if err != nil {
		return View{}, err
	}

	current, err := s.Get(ctx, vaID)
	if err != nil {
		return View{}, err
	}
	if current.Revision != expectedRevision {
		return View{}, ErrConflict
	}
	if DocumentsEqual(current.Document, doc) {
		return current, nil
	}

	var saved *Row
	err = s.lock.WithAdvisoryLock(ctx, vaID, func(tx *sql.Tx) error {
		again, err := s.Get(ctx, vaID)
		if err != nil {
			return err
		}
		if again.Revision != expectedRevision {
			return ErrConflict
		}
		if DocumentsEqual(again.Document, doc) {
			return nil
		}
		saved, err = s.repo.UpsertIfRevision(ctx, vaID, payload, expectedRevision, actor)
		_ = tx
		return err
	})
	if err != nil {
		return View{}, err
	}
	if saved == nil {
		return s.Get(ctx, vaID)
	}
	out, err := s.Get(ctx, vaID)
	if err != nil {
		return View{}, err
	}
	return out, nil
}

func (s *Service) ExportBytes(ctx context.Context, vaID string) ([]byte, error) {
	view, err := s.Get(ctx, vaID)
	if err != nil {
		return nil, err
	}
	return MarshalExport(view.Document)
}

type CatalogKind string

const (
	CatalogAircraft CatalogKind = "aircraft"
	CatalogAirline  CatalogKind = "airline"
)

func (s *Service) CatalogPage(ctx context.Context, vaID string, kind CatalogKind, search string, limit, offset int) ([]NameGroup, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	view, err := s.Get(ctx, vaID)
	if err != nil {
		return nil, 0, err
	}
	cfg := ConfigFromDocument(view.Document)
	kindStr := string(kind)
	if kind == CatalogAirline {
		kindStr = "airline"
	} else {
		kindStr = "aircraft"
	}
	rows, total, err := s.liveries.ListForCatalog(ctx, kindStr, strings.TrimSpace(search), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	all, err := s.liveries.ListAll(ctx)
	if err != nil {
		return nil, 0, err
	}
	names := BuildCatalogNames(all)
	var groups []NameGroup
	switch kind {
	case CatalogAirline:
		groups = airlineGroupsFromRows(rows, cfg, names)
	default:
		groups = aircraftGroupsFromRows(rows, cfg, names)
	}
	return groups, total, nil
}

func aircraftGroupsFromRows(rows []gameliveries.AircraftLivery, cfg Config, names CatalogNames) []NameGroup {
	seen := map[string]struct{}{}
	out := make([]NameGroup, 0, len(rows))
	for _, row := range rows {
		if row.AircraftName == "" {
			continue
		}
		if _, ok := seen[row.AircraftName]; ok {
			continue
		}
		seen[row.AircraftName] = struct{}{}
		_, inCatalog := names.Aircraft[row.AircraftName]
		st := RuleStatusForKey(row.AircraftName, cfg.AircraftMappings, cfg.DefaultAircraft, inCatalog)
		g := NameGroup{Name: row.AircraftName, Status: st}
		if v, ok := cfg.AircraftMappings[row.AircraftName]; ok {
			g.FilingValue = v
		} else if cfg.DefaultAircraft != nil {
			g.FilingValue = *cfg.DefaultAircraft
		}
		out = append(out, g)
	}
	return out
}

func airlineGroupsFromRows(rows []gameliveries.AircraftLivery, cfg Config, names CatalogNames) []NameGroup {
	seen := map[string]struct{}{}
	out := make([]NameGroup, 0, len(rows))
	for _, row := range rows {
		if row.LiveryName == "" {
			continue
		}
		if _, ok := seen[row.LiveryName]; ok {
			continue
		}
		seen[row.LiveryName] = struct{}{}
		_, inCatalog := names.Airline[row.LiveryName]
		st := RuleStatusForKey(row.LiveryName, cfg.AirlineMappings, cfg.DefaultAirline, inCatalog)
		g := NameGroup{Name: row.LiveryName, Status: st}
		if v, ok := cfg.AirlineMappings[row.LiveryName]; ok {
			g.FilingValue = v
		} else if cfg.DefaultAirline != nil {
			g.FilingValue = *cfg.DefaultAirline
		}
		out = append(out, g)
	}
	return out
}

func (s *Service) WriteCatalogCSV(ctx context.Context, w io.Writer) error {
	return s.liveries.WriteCatalogCSV(ctx, w)
}

func (s *Service) ResolveForVA(ctx context.Context, vaID string, input ResolveInput) (ResolveResult, error) {
	view, err := s.Get(ctx, vaID)
	if err != nil {
		return ResolveResult{}, err
	}
	cfg := ConfigFromDocument(view.Document)
	rows, err := s.liveries.ListAll(ctx)
	if err != nil {
		return ResolveResult{}, err
	}
	return s.resolveIDs(rows, cfg, input), nil
}
