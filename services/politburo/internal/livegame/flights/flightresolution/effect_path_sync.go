package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var pathSyncUpdateEffect = EffectDecision{
	Name:    "PathSyncUpdate",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyPathSyncUpdate,
}

func applyPathSyncUpdate(in EffectInput, out *flights.Derived, outcome *Outcome) {
	out.PathSync.FPLSyncRequired = in.Facts.FPLDue
	if in.Facts.FPLDue {
		out.PathSync.LastFPLSyncAt = in.Ctx.RefreshedAt
		outcome.FPLSyncScheduled = true
	}
}
