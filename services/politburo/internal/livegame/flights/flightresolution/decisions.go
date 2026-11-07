package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

// EffectDecision is one composable resolution step with separate criteria and action.
type EffectDecision struct {
	Name    string
	Applies func(in EffectInput) bool
	Apply   func(in EffectInput, analytics *flights.Analytics, actions *flights.Actions, links *flights.Links, outcome *Outcome)
}

var flightEffects = []EffectDecision{
	flightUsersEffect,
	flightStateEffect,
	vaMatchEffect,
	liveryResolveMetricEffect,
	fplSyncUpdateEffect,
}
