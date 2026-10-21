package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

// EffectDecision is one composable resolution step with separate criteria and action.
type EffectDecision struct {
	Name    string
	Applies func(in EffectInput) bool
	Apply   func(in EffectInput, out *flights.Derived, outcome *Outcome)
}

var flightEffects = []EffectDecision{
	normalizedDisplayEffect,
	flightStateEffect,
	vaMatchEffect,
	liveryEnrichEffect,
	pathSyncUpdateEffect,
}
