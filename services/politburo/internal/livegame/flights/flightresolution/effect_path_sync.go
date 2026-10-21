package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var pathSyncUpdateEffect = EffectDecision{
	Name:    "PathSyncUpdate",
	Applies: func(in EffectInput) bool { return in.Facts.FPLDue },
	Apply:   applyPathSyncUpdate,
}

func applyPathSyncUpdate(in EffectInput, out *flights.Derived, _ *Outcome) {
	out.PathSync.FPLSyncRequired = true
	out.PathSync.LastFPLSyncAt = in.Ctx.RefreshedAt
}
