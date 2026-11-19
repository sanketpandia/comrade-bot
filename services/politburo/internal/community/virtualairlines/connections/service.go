package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/secretstore"

	"github.com/google/uuid"
)

type blocksRepo interface {
	Get(ctx context.Context, vaID, configType string) (*vaconfig.Block, error)
	UpsertIfRevision(ctx context.Context, vaID, configType string, config json.RawMessage, expectedRevision int, updatedBy string) (*vaconfig.Block, error)
}

const airtablePATPurpose = "airtable.pat"

// Service owns configuration and live mapped reads.
type Service struct {
	repo     blocksRepo
	airtable *airtable.Client
	cipher   *secretstore.Cipher
	metrics  *metrics.Registry
	signKey  []byte
}

func NewService(repo blocksRepo, at *airtable.Client, cipher *secretstore.Cipher, cursorKey []byte, reg *metrics.Registry) (*Service, error) {
	if cipher != nil && len(cursorKey) != 32 {
		return nil, errors.New("connections: cursor key must be 32 bytes")
	}
	var signKey []byte
	if cipher != nil {
		signKey = bytes.Clone(cursorKey)
	}
	return &Service{repo: repo, airtable: at, cipher: cipher, metrics: reg, signKey: signKey}, nil
}

func (s *Service) Enabled() bool {
	return s != nil && s.cipher != nil
}

type StoredView struct {
	AdminView
	UpdatedAt time.Time
	UpdatedBy string
}

func (s *Service) GetView(ctx context.Context, vaID string) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, block, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	view := Redact(doc)
	return StoredView{AdminView: view, UpdatedAt: blockTime(block), UpdatedBy: blockActor(block)}, nil
}

func blockTime(block *vaconfig.Block) time.Time {
	if block == nil {
		return time.Time{}
	}
	return block.UpdatedAt
}

func blockActor(block *vaconfig.Block) string {
	if block == nil {
		return ""
	}
	return block.UpdatedBy
}

func (s *Service) load(ctx context.Context, vaID string) (Document, *vaconfig.Block, error) {
	block, err := s.repo.Get(ctx, vaID, vaconfig.TypeDataConnections)
	if errors.Is(err, vaconfig.ErrNotFound) {
		return EmptyDocument(), nil, nil
	}
	if err != nil {
		return Document{}, nil, err
	}
	doc, err := ParseDocument(block.Config)
	if err != nil {
		return Document{}, block, err
	}
	return doc, block, nil
}

func (s *Service) persist(ctx context.Context, vaID, actor string, expected int, doc Document) (*vaconfig.Block, error) {
	if err := ValidateDocument(doc); err != nil {
		return nil, err
	}
	raw, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	block, err := s.repo.UpsertIfRevision(ctx, vaID, vaconfig.TypeDataConnections, raw, expected, actor)
	if errors.Is(err, vaconfig.ErrConflict) {
		return nil, ErrConflict
	}
	return block, err
}

// CreateConnectionInput starts a new provider connection with one base.
type CreateConnectionInput struct {
	Name     string
	Provider string
	Token    string
	BaseName string
	BaseID   string
}

func (s *Service) CreateConnection(ctx context.Context, vaID, actor string, expectedRevision int, in CreateConnectionInput) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	if strings.TrimSpace(in.Token) == "" {
		return StoredView{}, ErrInvalidCredential
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	connID := uuid.NewString()
	baseID := uuid.NewString()
	if err := s.testAirtableAccess(ctx, in.Token, in.BaseID, ""); err != nil {
		return StoredView{}, err
	}
	enc, err := s.sealToken(vaID, connID, in.Token)
	if err != nil {
		return StoredView{}, err
	}
	conn := Connection{
		ID:       connID,
		Name:     strings.TrimSpace(in.Name),
		Provider: ProviderAirtable,
		Airtable: &AirtableSetup{
			Credential: enc,
			Bases: []BaseSetup{{
				ID:     baseID,
				Name:   strings.TrimSpace(in.BaseName),
				BaseID: strings.TrimSpace(in.BaseID),
			}},
		},
	}
	doc.Connections = append(doc.Connections, conn)
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

func (s *Service) expectRevision(doc Document, expected int) error {
	if doc.Revision != expected {
		return ErrConflict
	}
	return nil
}

func (s *Service) commit(ctx context.Context, vaID, actor string, expected int, doc Document) (StoredView, error) {
	old := doc.Revision
	doc.Revision++
	block, err := s.persist(ctx, vaID, actor, expected, doc)
	if err != nil {
		doc.Revision = old
		return StoredView{}, err
	}
	view := Redact(doc)
	return StoredView{
		AdminView: view,
		UpdatedAt: block.UpdatedAt,
		UpdatedBy: block.UpdatedBy,
	}, nil
}

func (s *Service) testAirtableAccess(ctx context.Context, token, baseID, tableID string) error {
	if _, err := s.fetchSchema(ctx, token, baseID); err != nil {
		return err
	}
	if tableID != "" {
		return s.probeTable(ctx, token, baseID, tableID)
	}
	return nil
}

func (s *Service) sealToken(vaID, connectionID, token string) (secretstore.Envelope, error) {
	if s == nil || s.cipher == nil {
		return secretstore.Envelope{}, ErrDisabled
	}
	return s.cipher.Encrypt(secretstore.Context{
		Purpose:  airtablePATPurpose,
		OwnerID:  vaID,
		RecordID: connectionID,
	}, []byte(token))
}

func (s *Service) tokenForConnection(vaID string, conn Connection) (string, error) {
	if s == nil || s.cipher == nil || conn.Airtable == nil || len(conn.Airtable.Credential.Sealed) == 0 {
		return "", ErrInvalidCredential
	}
	plain, err := s.cipher.Decrypt(secretstore.Context{
		Purpose:  airtablePATPurpose,
		OwnerID:  vaID,
		RecordID: conn.ID,
	}, conn.Airtable.Credential)
	if err != nil || len(plain) == 0 {
		secretstore.Wipe(plain)
		return "", ErrInvalidCredential
	}
	token := string(plain)
	secretstore.Wipe(plain)
	return token, nil
}

// TableDescriptor is live provider metadata for the dashboard.
type TableDescriptor struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Fields []FieldDescriptor `json:"fields"`
}

type FieldDescriptor struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	CompatibleKeys []string `json:"compatibleKeys"`
}

type InspectResult struct {
	Tables []TableDescriptor `json:"tables"`
}

func (s *Service) Inspect(ctx context.Context, vaID, connectionID, baseSetupID string) (InspectResult, error) {
	started := time.Now()
	defer func() { s.observeRead("inspect", "none", started, nil) }()
	if !s.Enabled() {
		return InspectResult{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		s.observeRead("inspect", "none", started, err)
		return InspectResult{}, err
	}
	conn, _, ok := connectionByID(doc, connectionID)
	if !ok {
		err = ErrNotFound
		s.observeRead("inspect", "none", started, err)
		return InspectResult{}, err
	}
	base, ok := baseByID(conn, baseSetupID)
	if !ok {
		err = ErrNotFound
		s.observeRead("inspect", "none", started, err)
		return InspectResult{}, err
	}
	token, err := s.tokenForConnection(vaID, conn)
	if err != nil {
		s.observeRead("inspect", "none", started, err)
		return InspectResult{}, err
	}
	tables, err := s.fetchSchema(ctx, token, base.BaseID)
	if err != nil {
		s.observeRead("inspect", "none", started, err)
		return InspectResult{}, err
	}
	out := InspectResult{Tables: make([]TableDescriptor, 0, len(tables))}
	for _, tbl := range tables {
		fields := make([]FieldDescriptor, 0, len(tbl.Fields))
		for _, f := range tbl.Fields {
			fields = append(fields, FieldDescriptor{
				ID:             f.ID,
				Name:           f.Name,
				Type:           f.Type,
				CompatibleKeys: compatibleKeysForType(f.Type),
			})
		}
		out.Tables = append(out.Tables, TableDescriptor{ID: tbl.ID, Name: tbl.Name, Fields: fields})
	}
	s.observeRead("inspect", "none", started, nil)
	return out, nil
}

func compatibleKeysForType(fieldType string) []string {
	keys := make([]string, 0)
	for entity, fields := range catalog {
		_ = entity
		for _, f := range fields {
			for _, t := range f.AcceptedTypes {
				if t == fieldType {
					keys = append(keys, f.Key)
					break
				}
			}
		}
	}
	return keys
}

func (s *Service) observeRead(operation, entity string, started time.Time, err error) {
	if s.metrics == nil {
		return
	}
	s.metrics.ObserveDataConnectionsRead(operation, entity, readOutcome(err), time.Since(started).Seconds())
}

func (s *Service) logProviderFailure(ctx context.Context, vaID, provider, operation, outcome string, status int, retry time.Duration, err error) {
	args := []any{
		"va_id", vaID,
		"provider", provider,
		"operation", operation,
		"outcome", outcome,
	}
	if status > 0 {
		args = append(args, "status", status)
	}
	if retry > 0 {
		args = append(args, "retry_after_ms", retry.Milliseconds())
	}
	if err != nil {
		args = append(args, "error", err)
	}
	slog.WarnContext(ctx, "data connection provider request failed", args...)
}

type ListResult struct {
	Records    []MappedRecord `json:"records"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

func (s *Service) List(ctx context.Context, vaID, entity string, filters map[string]any, pageSize int, cursor string) (ListResult, error) {
	started := time.Now()
	var err error
	defer func() { s.observeRead("list", entity, started, err) }()
	if !s.Enabled() {
		err = ErrDisabled
		return ListResult{}, err
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	doc, _, loadErr := s.load(ctx, vaID)
	if loadErr != nil {
		err = loadErr
		return ListResult{}, err
	}
	offset := ""
	if cursor != "" {
		offset, err = openCursor(s.signKey, cursor, vaID, entity, doc.Revision, filters)
		if err != nil {
			return ListResult{}, err
		}
	}
	mapping, ok := doc.Mappings[entity]
	if !ok {
		err = ErrMissingMapping
		return ListResult{}, err
	}
	conn, _, ok := connectionByID(doc, mapping.ConnectionID)
	if !ok {
		err = ErrMissingMapping
		return ListResult{}, err
	}
	base, ok := baseByID(conn, mapping.BaseSetupID)
	if !ok {
		err = ErrMissingMapping
		return ListResult{}, err
	}
	token, decErr := s.tokenForConnection(vaID, conn)
	if decErr != nil {
		err = decErr
		return ListResult{}, err
	}
	schema, schemaErr := s.schemaIndex(ctx, token, base.BaseID, mapping.TableID)
	if schemaErr != nil {
		err = schemaErr
		return ListResult{}, err
	}
	formula, formulaErr := buildFormula(entity, mapping, schema, filters)
	if formulaErr != nil {
		err = formulaErr
		return ListResult{}, err
	}
	names := fieldNamesForMapping(mapping, schema)
	raws, nextOffset, listErr := s.listRecords(ctx, token, base.BaseID, mapping.TableID, names, formula, pageSize, offset)
	if listErr != nil {
		err = listErr
		return ListResult{}, err
	}
	records := make([]MappedRecord, 0, len(raws))
	for _, raw := range raws {
		mapped, mapErr := NormalizeRecord(entity, mapping, schema, raw.Fields)
		if mapErr != nil {
			err = mapErr
			return ListResult{}, err
		}
		mapped.SourceID = raw.ID
		records = append(records, mapped)
	}
	nextCursor, curErr := buildCursorKey(s.signKey, vaID, entity, doc.Revision, filters, nextOffset)
	if curErr != nil {
		err = curErr
		return ListResult{}, err
	}
	return ListResult{Records: records, NextCursor: nextCursor}, nil
}

func (s *Service) schemaIndex(ctx context.Context, token, baseID, tableID string) (map[string]FieldMeta, error) {
	tables, err := s.fetchSchema(ctx, token, baseID)
	if err != nil {
		return nil, err
	}
	for _, tbl := range tables {
		if tbl.ID != tableID {
			continue
		}
		out := map[string]FieldMeta{}
		for _, f := range tbl.Fields {
			out[f.ID] = FieldMeta{ID: f.ID, Name: f.Name, Type: f.Type}
		}
		return out, nil
	}
	return nil, ErrSchemaDrift
}

func fieldNamesForMapping(mapping Mapping, schema map[string]FieldMeta) []string {
	names := make([]string, 0, len(mapping.Fields))
	seen := map[string]struct{}{}
	for _, fieldID := range mapping.Fields {
		meta, ok := schema[fieldID]
		if !ok {
			continue
		}
		if _, dup := seen[meta.Name]; dup {
			continue
		}
		seen[meta.Name] = struct{}{}
		names = append(names, meta.Name)
	}
	return names
}

func buildFormula(entity string, mapping Mapping, schema map[string]FieldMeta, filters map[string]any) (string, error) {
	if len(filters) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(filters))
	for key, value := range filters {
		fieldID, ok := mapping.Fields[key]
		if !ok {
			return "", fmt.Errorf("%w: unknown filter key %s", ErrInvalidFilter, key)
		}
		if _, ok := catalogField(entity, key); !ok {
			return "", ErrInvalidFilter
		}
		meta, ok := schema[fieldID]
		if !ok {
			return "", ErrSchemaDrift
		}
		lit, err := filterLiteralValue(value)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("%s='%s'", airtable.FieldRef(meta.Name), airtable.EscapeFormulaLiteral(lit)))
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return fmt.Sprintf("AND(%s)", strings.Join(parts, ",")), nil
}
