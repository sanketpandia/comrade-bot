package featureconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
)

type blocksRepo interface {
	Get(ctx context.Context, vaID, configType string) (*vaconfig.Block, error)
	UpsertIfRevision(ctx context.Context, vaID, configType string, config json.RawMessage, expectedRevision int, updatedBy string) (*vaconfig.Block, error)
	Delete(ctx context.Context, vaID, configType string) error
}

type dataConnections interface {
	Enabled() bool
	GetView(ctx context.Context, vaID string) (connections.StoredView, error)
}

// Service owns MAP_STATS and MAP_PIREP documents.
type Service struct {
	repo        blocksRepo
	lock        *vaconfig.Repository
	connections dataConnections
	preview     *Previewer
}

func NewService(repo blocksRepo, lock *vaconfig.Repository, conn dataConnections, preview *Previewer) *Service {
	return &Service{repo: repo, lock: lock, connections: conn, preview: preview}
}

func (s *Service) Enabled() bool {
	return s != nil && s.connections != nil && s.connections.Enabled()
}

type FeatureView struct {
	Feature              string                   `json:"feature"`
	Revision             int                      `json:"revision"`
	ConnectionsRevision  int                      `json:"connectionsRevision"`
	Enabled              bool                     `json:"enabled"`
	Configured           bool                     `json:"configured"`
	Catalog              []CatalogFieldForFeature `json:"catalog"`
	Mappings             map[string]connections.Mapping `json:"mappings,omitempty"`
	Stats                *StatsDocument           `json:"stats,omitempty"`
	PIREP                *PIREPDocument           `json:"pirep,omitempty"`
	UpdatedAt            time.Time                `json:"updatedAt"`
	UpdatedBy            string                   `json:"updatedBy"`
}

func (s *Service) Get(ctx context.Context, vaID, feature string) (FeatureView, error) {
	if !s.Enabled() {
		return FeatureView{}, connections.ErrDisabled
	}
	configType, err := ConfigTypeForFeature(feature)
	if err != nil {
		return FeatureView{}, err
	}
	connView, connErr := s.connections.GetView(ctx, vaID)
	if connErr != nil && !errors.Is(connErr, connections.ErrDisabled) {
		return FeatureView{}, connErr
	}
	view := FeatureView{
		Feature: feature,
		Catalog: catalogForFeature(feature),
	}
	if connErr == nil {
		view.ConnectionsRevision = connView.Revision
		view.Mappings = connView.Mappings
	}
	block, err := s.repo.Get(ctx, vaID, configType)
	if errors.Is(err, vaconfig.ErrNotFound) {
		return view, nil
	}
	if err != nil {
		return FeatureView{}, err
	}
	view.UpdatedAt = block.UpdatedAt
	view.UpdatedBy = block.UpdatedBy
	switch feature {
	case FeatureStats:
		doc, err := ParseStatsDocument(block.Config)
		if err != nil {
			return FeatureView{}, err
		}
		view.Revision = doc.Revision
		view.Enabled = doc.Enabled
		view.Configured = doc.Pilot != nil || doc.Career != nil
		view.Stats = &doc
	case FeaturePIREP:
		doc, err := ParsePIREPDocument(block.Config)
		if err != nil {
			return FeatureView{}, err
		}
		view.Revision = doc.Revision
		view.Enabled = doc.Enabled
		view.Configured = len(doc.Fields) > 0
		view.PIREP = &doc
	}
	return view, nil
}

func catalogForFeature(feature string) []CatalogFieldForFeature {
	switch feature {
	case FeatureStats:
		return StatsCatalogFields()
	case FeaturePIREP:
		return PIREPCatalogFields()
	default:
		return nil
	}
}

func (s *Service) SaveStats(ctx context.Context, vaID, actor string, expected, connectionsRevision int, doc StatsDocument) (FeatureView, error) {
	if !s.Enabled() {
		return FeatureView{}, connections.ErrDisabled
	}
	connView, err := s.connections.GetView(ctx, vaID)
	if err != nil {
		return FeatureView{}, err
	}
	if connView.Revision != connectionsRevision {
		return FeatureView{}, ErrStaleConnections
	}
	if err := ValidateStatsDraft(doc, connView.Mappings); err != nil {
		return FeatureView{}, err
	}
	doc.Enabled = doc.Pilot != nil || doc.Career != nil
	doc.Revision++
	raw, err := doc.Marshal()
	if err != nil {
		return FeatureView{}, err
	}
	persist := func() error {
		_, err := s.repo.UpsertIfRevision(ctx, vaID, vaconfig.TypeMapStats, raw, expected, actor)
		if errors.Is(err, vaconfig.ErrConflict) {
			return ErrConflict
		}
		return err
	}
	if s.lock != nil {
		if err := s.lock.WithAdvisoryLock(ctx, vaID, func(tx *sql.Tx) error { return persist() }); err != nil {
			return FeatureView{}, err
		}
	} else if err := persist(); err != nil {
		return FeatureView{}, err
	}
	return s.Get(ctx, vaID, FeatureStats)
}

func (s *Service) SavePIREP(ctx context.Context, vaID, actor string, expected, connectionsRevision int, doc PIREPDocument) (FeatureView, error) {
	if !s.Enabled() {
		return FeatureView{}, connections.ErrDisabled
	}
	connView, err := s.connections.GetView(ctx, vaID)
	if err != nil {
		return FeatureView{}, err
	}
	if connView.Revision != connectionsRevision {
		return FeatureView{}, ErrStaleConnections
	}
	if err := ValidatePIREPDraft(doc, connView.Mappings); err != nil {
		return FeatureView{}, err
	}
	doc.Enabled = len(doc.Fields) > 0
	doc.Revision++
	raw, err := doc.Marshal()
	if err != nil {
		return FeatureView{}, err
	}
	persist := func() error {
		_, err := s.repo.UpsertIfRevision(ctx, vaID, vaconfig.TypeMapPIREP, raw, expected, actor)
		if errors.Is(err, vaconfig.ErrConflict) {
			return ErrConflict
		}
		return err
	}
	if s.lock != nil {
		if err := s.lock.WithAdvisoryLock(ctx, vaID, func(tx *sql.Tx) error { return persist() }); err != nil {
			return FeatureView{}, err
		}
	} else if err := persist(); err != nil {
		return FeatureView{}, err
	}
	return s.Get(ctx, vaID, FeaturePIREP)
}

func (s *Service) Delete(ctx context.Context, vaID, feature string, expected int) error {
	if !s.Enabled() {
		return connections.ErrDisabled
	}
	configType, err := ConfigTypeForFeature(feature)
	if err != nil {
		return err
	}
	block, err := s.repo.Get(ctx, vaID, configType)
	if errors.Is(err, vaconfig.ErrNotFound) {
		return vaconfig.ErrNotFound
	}
	if err != nil {
		return err
	}
	docRev := 0
	switch feature {
	case FeatureStats:
		doc, err := ParseStatsDocument(block.Config)
		if err != nil {
			return err
		}
		docRev = doc.Revision
	case FeaturePIREP:
		doc, err := ParsePIREPDocument(block.Config)
		if err != nil {
			return err
		}
		docRev = doc.Revision
	}
	if docRev != expected {
		return ErrConflict
	}
	return s.repo.Delete(ctx, vaID, configType)
}

type PreviewInput struct {
	Callsign     string `json:"callsign,omitempty"`
	IFCUsername  string `json:"ifcUsername,omitempty"`
	Stats        *StatsDocument `json:"stats,omitempty"`
	PIREP        *PIREPDocument `json:"pirep,omitempty"`
}

type PreviewResult struct {
	Disabled  bool           `json:"disabled,omitempty"`
	Sections  []PreviewSection `json:"sections,omitempty"`
	Warnings  []string       `json:"warnings,omitempty"`
	Filing    []FilingSample `json:"filing,omitempty"`
}

type PreviewSection struct {
	Name   string         `json:"name"`
	Fields []PreviewField `json:"fields"`
}

type PreviewField struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Value         any    `json:"value,omitempty"`
	Display       string `json:"display,omitempty"`
	Unresolved    bool   `json:"unresolved,omitempty"`
	Warning       string `json:"warning,omitempty"`
}

type FilingSample struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	Destination   string `json:"destinationColumn,omitempty"`
	WriteValue    any    `json:"writeValue,omitempty"`
	Display       string `json:"display,omitempty"`
	Synthetic     bool   `json:"synthetic"`
}

func (s *Service) Preview(ctx context.Context, vaID, feature string, in PreviewInput) (PreviewResult, error) {
	if !s.Enabled() {
		return PreviewResult{}, connections.ErrDisabled
	}
	if s.preview == nil {
		return PreviewResult{}, errors.New("feature preview not configured")
	}
	switch feature {
	case FeatureStats:
		doc := in.Stats
		if doc == nil {
			got, err := s.Get(ctx, vaID, FeatureStats)
			if err != nil {
				return PreviewResult{}, err
			}
			doc = got.Stats
		}
		if doc == nil || !doc.Enabled {
			return PreviewResult{Disabled: true}, ErrDisabled
		}
		return s.preview.Stats(ctx, vaID, *doc, in.Callsign, in.IFCUsername)
	case FeaturePIREP:
		doc := in.PIREP
		if doc == nil {
			got, err := s.Get(ctx, vaID, FeaturePIREP)
			if err != nil {
				return PreviewResult{}, err
			}
			doc = got.PIREP
		}
		if doc == nil || !doc.Enabled {
			return PreviewResult{Disabled: true}, ErrDisabled
		}
		return s.preview.PIREP(ctx, vaID, *doc)
	default:
		return PreviewResult{}, ErrInvalidFeature
	}
}

// LoadStatsForRuntime returns the cached stats document when enabled.
func (s *Service) LoadStatsForRuntime(ctx context.Context, vaID string) (StatsDocument, error) {
	block, err := s.repo.Get(ctx, vaID, vaconfig.TypeMapStats)
	if errors.Is(err, vaconfig.ErrNotFound) {
		return StatsDocument{}, ErrNotConfigured
	}
	if err != nil {
		return StatsDocument{}, err
	}
	doc, err := ParseStatsDocument(block.Config)
	if err != nil {
		return StatsDocument{}, err
	}
	if !doc.Enabled {
		return StatsDocument{}, ErrDisabled
	}
	return doc, nil
}