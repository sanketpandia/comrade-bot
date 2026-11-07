package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var fplSyncUpdateEffect = EffectDecision{
	Name:    "FPLSyncUpdate",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyFPLSyncUpdate,
}

func applyFPLSyncUpdate(in EffectInput, _ *flights.Analytics, actions *flights.Actions, _ *flights.Links, outcome *Outcome) {
	actions.FPLSync.Required = in.Facts.FPLDue
	if in.Facts.FPLDue {
		actions.FPLSync.LastAt = in.Ctx.RefreshedAt
		outcome.FPLSyncScheduled = true
	}
}
