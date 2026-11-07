package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var vaMatchEffect = EffectDecision{
	Name:    "VAMatch",
	Applies: func(in EffectInput) bool { return true },
	Apply:   applyVAMatch,
}

func applyVAMatch(in EffectInput, _ *flights.Analytics, _ *flights.Actions, links *flights.Links, _ *Outcome) {
	_, matches := callsignAffixFields(in)
	if flights.VirtualAirlineMatchesEqual(links.VirtualAirlines, matches) {
		return
	}
	links.VirtualAirlines = matches
}
