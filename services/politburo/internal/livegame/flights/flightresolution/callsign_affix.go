package flightresolution

import (
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
)

func callsignAffixFields(in EffectInput) (string, []virtualairlines.VirtualAirlineMatch) {
	callsign := in.Raw.Callsign
	normalized := flights.NormalizedCallsignForAffix(callsign)
	return normalized, in.Ctx.LiveVACatalog.Match(callsign, normalized, in.Ctx.Session.NormalizedName)
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
