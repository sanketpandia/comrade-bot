package flights

import (
	"reflect"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

// Record is the per-flight cache entry: immutable upstream observation plus derived enrichment.
type Record struct {
	Raw     Observation `json:"raw"`
	Derived Derived     `json:"derived"`
}

// Derived holds effect-owned enrichment separate from upstream observation.
type Derived struct {
	NormalizedCallsign     string                                `json:"normalizedCallsign,omitempty"`
	AircraftName           string                                `json:"aircraftName,omitempty"`
	LiveryName             string                                `json:"liveryName,omitempty"`
	Normalized             Normalized                            `json:"normalized"`
	PathSync               PathSync                              `json:"pathSync"`
	MatchedVirtualAirlines []virtualairlines.VirtualAirlineMatch `json:"matchedVirtualAirlines,omitempty"`
	FlightState            *flightstate.Summary                  `json:"flightState,omitempty"`
}

// Clone returns a deep copy of derived fields.
func (d Derived) Clone() Derived {
	out := d
	if len(d.MatchedVirtualAirlines) > 0 {
		out.MatchedVirtualAirlines = append([]virtualairlines.VirtualAirlineMatch(nil), d.MatchedVirtualAirlines...)
	}
	if d.FlightState != nil {
		out.FlightState = &flightstate.Summary{}
		*out.FlightState = *d.FlightState
	}
	return out
}

// Equal reports whether two derived snapshots are identical for cache dirty checks.
func (d Derived) Equal(other Derived) bool {
	return d.NormalizedCallsign == other.NormalizedCallsign &&
		d.AircraftName == other.AircraftName &&
		d.LiveryName == other.LiveryName &&
		d.Normalized == other.Normalized &&
		d.PathSync == other.PathSync &&
		virtualAirlineMatchesEqual(d.MatchedVirtualAirlines, other.MatchedVirtualAirlines) &&
		flightStateEqual(d.FlightState, other.FlightState)
}

func flightStateEqual(a, b *flightstate.Summary) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func virtualAirlineMatchesEqual(a, b []virtualairlines.VirtualAirlineMatch) bool {
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
