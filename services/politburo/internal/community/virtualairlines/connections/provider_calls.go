package connections

import (
	"context"
	"errors"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

func (s *Service) fetchSchema(ctx context.Context, token, baseID string) ([]airtable.Table, error) {
	tables, err := s.airtable.FetchSchema(ctx, token, baseID)
	return tables, noteProviderCall(mapAirtableError(err), "schema", baseID)
}

func (s *Service) probeTable(ctx context.Context, token, baseID, tableID string) error {
	return noteProviderCall(mapAirtableError(s.airtable.ProbeTable(ctx, token, baseID, tableID)), "probe", baseID)
}

func (s *Service) listRecords(ctx context.Context, token, baseID, tableID string, fieldNames []string, formula string, pageSize int, offset string) ([]airtable.RawRecord, string, error) {
	raws, next, err := s.airtable.ListRecords(ctx, token, baseID, tableID, fieldNames, formula, pageSize, offset)
	return raws, next, noteProviderCall(mapAirtableError(err), "list", baseID)
}

func (s *Service) getRecord(ctx context.Context, token, baseID, tableID, recordID string, fieldNames []string) (airtable.RawRecord, error) {
	raw, err := s.airtable.GetRecord(ctx, token, baseID, tableID, recordID, fieldNames)
	return raw, noteProviderCall(mapAirtableError(err), "get", baseID)
}

func noteProviderCall(err error, operation, baseID string) error {
	var pe *ProviderError
	if errors.As(err, &pe) {
		pe.Operation = operation
		pe.BaseID = baseID
	}
	return err
}
