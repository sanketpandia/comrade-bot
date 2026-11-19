package connections

import (
	"context"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"

	"github.com/google/uuid"
)

type UpdateConnectionInput struct {
	Name  *string
	Token *string
}

func (s *Service) UpdateConnection(ctx context.Context, vaID, actor, connectionID string, expectedRevision int, in UpdateConnectionInput) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	conn, idx, ok := connectionByID(doc, connectionID)
	if !ok {
		return StoredView{}, ErrNotFound
	}
	if in.Name != nil {
		conn.Name = strings.TrimSpace(*in.Name)
	}
	if in.Token != nil {
		token := strings.TrimSpace(*in.Token)
		if token == "" {
			return StoredView{}, ErrInvalidCredential
		}
		for _, base := range conn.Airtable.Bases {
			if err := s.testAirtableAccess(ctx, token, base.BaseID, ""); err != nil {
				return StoredView{}, err
			}
		}
		enc, err := s.cipher.Encrypt(vaID, conn.ID, token)
		if err != nil {
			return StoredView{}, err
		}
		conn.Airtable.Credential = enc
	}
	doc.Connections[idx] = conn
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

func (s *Service) DeleteConnection(ctx context.Context, vaID, actor, connectionID string, expectedRevision int) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	if mappingReferencesConnection(doc, connectionID) {
		return StoredView{}, ErrInUse
	}
	out := make([]Connection, 0, len(doc.Connections))
	var found bool
	for _, c := range doc.Connections {
		if c.ID == connectionID {
			found = true
			continue
		}
		out = append(out, c)
	}
	if !found {
		return StoredView{}, ErrNotFound
	}
	doc.Connections = out
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

type AddBaseInput struct {
	Name   string
	BaseID string
}

func (s *Service) AddBase(ctx context.Context, vaID, actor, connectionID string, expectedRevision int, in AddBaseInput) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	conn, idx, ok := connectionByID(doc, connectionID)
	if !ok || conn.Airtable == nil {
		return StoredView{}, ErrNotFound
	}
	token, err := s.tokenForConnection(vaID, conn)
	if err != nil {
		return StoredView{}, err
	}
	baseID := strings.TrimSpace(in.BaseID)
	if err := s.testAirtableAccess(ctx, token, baseID, ""); err != nil {
		return StoredView{}, err
	}
	conn.Airtable.Bases = append(conn.Airtable.Bases, BaseSetup{
		ID:     uuid.NewString(),
		Name:   strings.TrimSpace(in.Name),
		BaseID: baseID,
	})
	doc.Connections[idx] = conn
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

type UpdateBaseInput struct {
	Name   *string
	BaseID *string
}

func (s *Service) UpdateBase(ctx context.Context, vaID, actor, connectionID, baseSetupID string, expectedRevision int, in UpdateBaseInput) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	conn, connIdx, ok := connectionByID(doc, connectionID)
	if !ok || conn.Airtable == nil {
		return StoredView{}, ErrNotFound
	}
	var baseIdx = -1
	for i, b := range conn.Airtable.Bases {
		if b.ID == baseSetupID {
			baseIdx = i
			break
		}
	}
	if baseIdx < 0 {
		return StoredView{}, ErrNotFound
	}
	base := conn.Airtable.Bases[baseIdx]
	if in.Name != nil {
		base.Name = strings.TrimSpace(*in.Name)
	}
	if in.BaseID != nil && strings.TrimSpace(*in.BaseID) != base.BaseID {
		return StoredView{}, ErrBaseIDImmutable
	}
	conn.Airtable.Bases[baseIdx] = base
	doc.Connections[connIdx] = conn
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

func (s *Service) DeleteBase(ctx context.Context, vaID, actor, connectionID, baseSetupID string, expectedRevision int) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	if mappingReferencesBase(doc, baseSetupID) {
		return StoredView{}, ErrInUse
	}
	conn, connIdx, ok := connectionByID(doc, connectionID)
	if !ok || conn.Airtable == nil {
		return StoredView{}, ErrNotFound
	}
	bases := make([]BaseSetup, 0, len(conn.Airtable.Bases))
	var found bool
	for _, b := range conn.Airtable.Bases {
		if b.ID == baseSetupID {
			found = true
			continue
		}
		bases = append(bases, b)
	}
	if !found {
		return StoredView{}, ErrNotFound
	}
	conn.Airtable.Bases = bases
	doc.Connections[connIdx] = conn
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

func (s *Service) SaveMapping(ctx context.Context, vaID, actor, entity string, expectedRevision int, mapping Mapping) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	if entity != EntityPilots && entity != EntityRoutes && entity != EntityPIREPs {
		return StoredView{}, ErrInvalidMapping
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	conn, _, ok := connectionByID(doc, mapping.ConnectionID)
	if !ok {
		return StoredView{}, ErrNotFound
	}
	base, ok := baseByID(conn, mapping.BaseSetupID)
	if !ok {
		return StoredView{}, ErrNotFound
	}
	if !MappingMeetsMinimum(entity, mapping.Fields) {
		return StoredView{}, ErrInvalidMapping
	}
	token, err := s.tokenForConnection(vaID, conn)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.validateMappingLive(ctx, token, base.BaseID, entity, mapping); err != nil {
		return StoredView{}, err
	}
	if doc.Mappings == nil {
		doc.Mappings = map[string]Mapping{}
	}
	doc.Mappings[entity] = mapping
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

func (s *Service) ClearMapping(ctx context.Context, vaID, actor, entity string, expectedRevision int) (StoredView, error) {
	if !s.Enabled() {
		return StoredView{}, ErrDisabled
	}
	doc, _, err := s.load(ctx, vaID)
	if err != nil {
		return StoredView{}, err
	}
	if err := s.expectRevision(doc, expectedRevision); err != nil {
		return StoredView{}, err
	}
	delete(doc.Mappings, entity)
	return s.commit(ctx, vaID, actor, expectedRevision, doc)
}

func (s *Service) Preview(ctx context.Context, vaID, entity string, mapping Mapping) ([]MappedRecord, error) {
	started := time.Now()
	var err error
	defer func() { s.observeRead("preview", entity, started, err) }()
	if !s.Enabled() {
		err = ErrDisabled
		return nil, err
	}
	if !MappingMeetsMinimum(entity, mapping.Fields) {
		err = ErrInvalidMapping
		return nil, err
	}
	doc, _, loadErr := s.load(ctx, vaID)
	if loadErr != nil {
		err = loadErr
		return nil, err
	}
	conn, _, ok := connectionByID(doc, mapping.ConnectionID)
	if !ok {
		err = ErrNotFound
		return nil, err
	}
	base, ok := baseByID(conn, mapping.BaseSetupID)
	if !ok {
		err = ErrNotFound
		return nil, err
	}
	token, decErr := s.tokenForConnection(vaID, conn)
	if decErr != nil {
		err = decErr
		return nil, err
	}
	if err = s.validateMappingLive(ctx, token, base.BaseID, entity, mapping); err != nil {
		return nil, err
	}
	schema, schemaErr := s.schemaIndex(ctx, token, base.BaseID, mapping.TableID)
	if schemaErr != nil {
		err = schemaErr
		return nil, err
	}
	names := fieldNamesForMapping(mapping, schema)
	raws, _, listErr := s.listRecords(ctx, token, base.BaseID, mapping.TableID, names, "", 5, "")
	if listErr != nil {
		err = listErr
		return nil, err
	}
	out := make([]MappedRecord, 0, len(raws))
	for _, raw := range raws {
		mapped, mapErr := NormalizeRecord(entity, mapping, schema, raw.Fields)
		if mapErr != nil {
			err = mapErr
			return nil, err
		}
		mapped.SourceID = raw.ID
		out = append(out, mapped)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, vaID, entity, sourceID string) (MappedRecord, error) {
	started := time.Now()
	var err error
	defer func() { s.observeRead("get", entity, started, err) }()
	if !s.Enabled() {
		err = ErrDisabled
		return MappedRecord{}, err
	}
	doc, _, loadErr := s.load(ctx, vaID)
	if loadErr != nil {
		err = loadErr
		return MappedRecord{}, err
	}
	mapping, ok := doc.Mappings[entity]
	if !ok {
		err = ErrMissingMapping
		return MappedRecord{}, err
	}
	conn, _, ok := connectionByID(doc, mapping.ConnectionID)
	if !ok {
		err = ErrMissingMapping
		return MappedRecord{}, err
	}
	base, ok := baseByID(conn, mapping.BaseSetupID)
	if !ok {
		err = ErrMissingMapping
		return MappedRecord{}, err
	}
	token, decErr := s.tokenForConnection(vaID, conn)
	if decErr != nil {
		err = decErr
		return MappedRecord{}, err
	}
	schema, schemaErr := s.schemaIndex(ctx, token, base.BaseID, mapping.TableID)
	if schemaErr != nil {
		err = schemaErr
		return MappedRecord{}, err
	}
	names := fieldNamesForMapping(mapping, schema)
	raw, getErr := s.getRecord(ctx, token, base.BaseID, mapping.TableID, sourceID, names)
	if getErr != nil {
		err = getErr
		return MappedRecord{}, err
	}
	mapped, mapErr := NormalizeRecord(entity, mapping, schema, raw.Fields)
	if mapErr != nil {
		err = mapErr
		return MappedRecord{}, err
	}
	mapped.SourceID = raw.ID
	return mapped, nil
}

func (s *Service) validateMappingLive(ctx context.Context, token, baseID, entity string, mapping Mapping) error {
	tables, err := s.fetchSchema(ctx, token, baseID)
	if err != nil {
		return err
	}
	var table *airtable.Table
	for i := range tables {
		if tables[i].ID == mapping.TableID {
			table = &tables[i]
			break
		}
	}
	if table == nil {
		return ErrSchemaDrift
	}
	byID := map[string]airtable.Field{}
	for _, f := range table.Fields {
		byID[f.ID] = f
	}
	for logicalKey, fieldID := range mapping.Fields {
		cf, ok := catalogField(entity, logicalKey)
		if !ok {
			return ErrInvalidMapping
		}
		f, ok := byID[fieldID]
		if !ok {
			return ErrSchemaDrift
		}
		if !typeCompatible(cf, f.Type) {
			return ErrInvalidMapping
		}
	}
	return s.probeTable(ctx, token, baseID, mapping.TableID)
}

func typeCompatible(cf CatalogField, providerType string) bool {
	for _, t := range cf.AcceptedTypes {
		if t == providerType {
			return true
		}
	}
	return false
}