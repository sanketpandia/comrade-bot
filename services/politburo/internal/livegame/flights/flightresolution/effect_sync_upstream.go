package flightresolution

import (
	"infinite-experiment/politburo/internal/livegame/flights"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

var syncFromUpstreamEffect = EffectDecision{
	Name:    "SyncFromUpstream",
	Applies: syncFromUpstreamApplies,
	Apply:   applySyncFromUpstream,
}

func syncFromUpstreamApplies(env *resolveEnv) bool {
	if env.prior == nil {
		return true
	}
	if !env.ctx.HasPriorMotion {
		return true
	}
	if !flights.MotionEqual(env.motion, env.ctx.PriorMotion) {
		return true
	}
	return env.fplDue
}

func applySyncFromUpstream(env *resolveEnv) {
	var resolved *gameliveries.ResolvedNames
	if liveryResolveNeeded(env.prior, env.upstream.AircraftID, env.upstream.LiveryID) {
		resolved = lookupLiveryNames(env)
	}
	env.flight = flights.MapFlight(env.upstream, env.ctx.Session, resolved, env.prior, env.ctx.RefreshedAt)
	env.hasFlight = true
	env.syncedUpstream = true
	env.recordMutated = true
}
