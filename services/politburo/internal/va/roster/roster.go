package roster

import "context"

// LookupResult is an optional Airtable pilots row pointer.
type LookupResult struct {
	AirtablePilotID string
}

// Checker validates VA roster membership by IFC username when pilots sync is enabled.
type Checker interface {
	LookupPilotByIFC(ctx context.Context, vaID, ifcUsername string) (*LookupResult, error)
}

// Noop skips roster validation until the Airtable mapper is wired.
type Noop struct{}

func (Noop) LookupPilotByIFC(context.Context, string, string) (*LookupResult, error) {
	return nil, nil
}
