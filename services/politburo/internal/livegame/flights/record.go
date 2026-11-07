package flights

import (
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

// Record is the per-flight cache entry: upstream observation plus effect-owned sections.
type Record struct {
	Raw         Observation      `json:"raw"`
	Analytics   Analytics        `json:"analytics"`
	Actions     Actions          `json:"actions"`
	Links       Links            `json:"links"`
	History     flightstate.History `json:"history"`
	Normalized  RecordNormalized `json:"normalized"`
}

// RecordNormalized holds normalized fields persisted from background sync (for example route endpoints from a cached flight plan).
type RecordNormalized struct {
	Departure   string `json:"departure,omitempty"`
	Destination string `json:"destination,omitempty"`
}

// Analytics holds temporal estimation and other derived analysis stored on the record.
type Analytics struct {
	FlightState *flightstate.Summary `json:"flightState,omitempty"`
}

// Actions holds job-oriented decisions computed during sync.
type Actions struct {
	FPLSync FPLSyncAction `json:"fplSync"`
}

// FPLSyncAction tracks flight-plan sync scheduling state on the cached record.
type FPLSyncAction struct {
	Required bool      `json:"required"`
	LastAt   time.Time `json:"lastAt,omitempty"`
}

// Links holds associations to other domains (VAs, operators).
type Links struct {
	VirtualAirlines []virtualairlines.VirtualAirlineMatch `json:"virtualAirlines,omitempty"`
	Users           []string                              `json:"users,omitempty"`
}

// Normalized is the serve-time display envelope (motion fields are derived from raw; route endpoints come from the cached record).
type Normalized struct {
	NormalizedCallsign string `json:"normalizedCallsign,omitempty"`
	AircraftName       string `json:"aircraftName,omitempty"`
	LiveryName         string `json:"liveryName,omitempty"`
	Departure          string `json:"departure,omitempty"`
	Destination        string `json:"destination,omitempty"`
	Speed              string `json:"speed"`
	VerticalSpeed      string `json:"verticalSpeed"`
	PilotState         string `json:"pilotState"`
	IsConnected        string `json:"isConnected"`
}

// StoredSectionsEqual compares cache-persisted sections for dirty checks (raw is always refreshed).
func (r Record) StoredSectionsEqual(other Record) bool {
	return r.Analytics.Equal(other.Analytics) &&
		r.Actions.Equal(other.Actions) &&
		r.Links.Equal(other.Links) &&
		r.History.Equal(other.History) &&
		r.Normalized.Equal(other.Normalized)
}

// Equal reports whether two stored normalized snapshots are identical for cache dirty checks.
func (n RecordNormalized) Equal(other RecordNormalized) bool {
	return n == other
}

// Clone returns a deep copy of analytics.
func (a Analytics) Clone() Analytics {
	out := a
	if a.FlightState != nil {
		cloned := a.FlightState.Clone()
		out.FlightState = &cloned
	}
	return out
}

// Equal reports whether two analytics snapshots are identical for cache dirty checks.
func (a Analytics) Equal(other Analytics) bool {
	return flightStateEqual(a.FlightState, other.FlightState)
}

// Clone returns a deep copy of actions.
func (ac Actions) Clone() Actions {
	return ac
}

// Equal reports whether two action snapshots are identical for cache dirty checks.
func (ac Actions) Equal(other Actions) bool {
	return ac.FPLSync == other.FPLSync
}

// Clone returns a deep copy of links.
func (l Links) Clone() Links {
	out := Links{}
	if len(l.VirtualAirlines) > 0 {
		out.VirtualAirlines = append([]virtualairlines.VirtualAirlineMatch(nil), l.VirtualAirlines...)
	}
	if len(l.Users) > 0 {
		out.Users = append([]string(nil), l.Users...)
	}
	return out
}

// Equal reports whether two link snapshots are identical for cache dirty checks.
func (l Links) Equal(other Links) bool {
	return VirtualAirlineMatchesEqual(l.VirtualAirlines, other.VirtualAirlines) &&
		UserIDsEqual(l.Users, other.Users)
}

func flightStateEqual(a, b *flightstate.Summary) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

// VirtualAirlineMatchesEqual reports whether two VA match slices are identical.
func VirtualAirlineMatchesEqual(a, b []virtualairlines.VirtualAirlineMatch) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
