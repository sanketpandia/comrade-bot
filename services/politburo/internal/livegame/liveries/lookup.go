package liveries

import (
	"context"
	"sync"
)

// MatchOutcome classifies how a flight was matched to the livery catalog.
type MatchOutcome string

const (
	MatchLivery       MatchOutcome = "livery"
	MatchAircraftOnly MatchOutcome = "aircraft_only"
	MatchMiss         MatchOutcome = "miss"
)

// ResolvedNames are display strings attached to a cached flight.
type ResolvedNames struct {
	AircraftName string
	LiveryName   string
}

type Lookup struct {
	repo *Repository

	mu           sync.RWMutex
	byLiveryID   map[string]AircraftLivery
	byAircraftID map[string]AircraftLivery
}

func NewLookup(repo *Repository) *Lookup {
	return &Lookup{repo: repo, byLiveryID: map[string]AircraftLivery{}, byAircraftID: map[string]AircraftLivery{}}
}

// NewStaticLookup builds an in-memory-only lookup (used in tests).
func NewStaticLookup(byLivery map[string]AircraftLivery, byAircraft map[string]AircraftLivery) *Lookup {
	if byLivery == nil {
		byLivery = map[string]AircraftLivery{}
	}
	if byAircraft == nil {
		byAircraft = map[string]AircraftLivery{}
	}
	return &Lookup{byLiveryID: byLivery, byAircraftID: byAircraft}
}

func (l *Lookup) Reload(ctx context.Context) error {
	rows, err := l.repo.ListAll(ctx)
	if err != nil {
		return err
	}
	byLivery := make(map[string]AircraftLivery, len(rows))
	byAircraft := make(map[string]AircraftLivery, len(rows))
	for _, row := range rows {
		byLivery[row.LiveryID] = row
		if row.AircraftID == "" {
			continue
		}
		current, exists := byAircraft[row.AircraftID]
		if !exists || row.UpdatedAt.After(current.UpdatedAt) {
			byAircraft[row.AircraftID] = row
		}
	}
	l.mu.Lock()
	l.byLiveryID = byLivery
	l.byAircraftID = byAircraft
	l.mu.Unlock()
	return nil
}

func (l *Lookup) Resolve(liveryID, aircraftID string) (ResolvedNames, MatchOutcome) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if liveryID != "" {
		if row, ok := l.byLiveryID[liveryID]; ok {
			return ResolvedNames{
				AircraftName: row.DisplayAircraftName,
				LiveryName:   row.DisplayLiveryName,
			}, MatchLivery
		}
	}
	if aircraftID != "" {
		if row, ok := l.byAircraftID[aircraftID]; ok {
			name := row.DisplayAircraftName
			if name == "" {
				name = row.AircraftName
			}
			return ResolvedNames{AircraftName: name, LiveryName: UnrecognizedLiveryName}, MatchAircraftOnly
		}
	}
	return ResolvedNames{}, MatchMiss
}
