package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var vaMatchEffect = EffectDecision{
	Name:    "VAMatch",
	Applies: func(in EffectInput) bool { return true },
	Apply:   applyVAMatch,
}

func applyVAMatch(in EffectInput, out *flights.Derived, _ *Outcome) {
	normalized, matches := callsignAffixFields(in)
	if out.NormalizedCallsign == normalized && virtualAirlineMatchesEqual(out.MatchedVirtualAirlines, matches) {
		return
	}
	out.NormalizedCallsign = normalized
	out.MatchedVirtualAirlines = matches
}
