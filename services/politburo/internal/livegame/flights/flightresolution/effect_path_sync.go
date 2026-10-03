package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var pathSyncUpdateEffect = EffectDecision{
	Name:    "PathSyncUpdate",
	Applies: pathSyncUpdateApplies,
	Apply:   applyPathSyncUpdate,
}

func pathSyncUpdateApplies(env *resolveEnv) bool {
	return env.fplDue || env.syncedUpstream
}

func applyPathSyncUpdate(env *resolveEnv) {
	if !env.hasFlight {
		return
	}
	pathSync := env.flight.PathSync
	if pathSync == nil {
		pathSync = &flights.PathSync{}
		env.flight.PathSync = pathSync
	}
	pathSync.FPLSyncRequired = env.fplDue
	if env.fplDue {
		pathSync.LastFPLSyncAt = env.ctx.RefreshedAt
	} else if env.prior != nil && env.prior.PathSync != nil && !env.prior.PathSync.LastFPLSyncAt.IsZero() {
		pathSync.LastFPLSyncAt = env.prior.PathSync.LastFPLSyncAt
	}
	env.recordMutated = true
}
