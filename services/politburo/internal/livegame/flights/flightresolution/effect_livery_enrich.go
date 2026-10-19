package flightresolution

import (
	"infinite-experiment/politburo/internal/livegame/flights"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

var liveryEnrichEffect = EffectDecision{
	Name:    "LiveryEnrich",
	Applies: liveryEnrichApplies,
	Apply:   applyLiveryEnrich,
}

func liveryEnrichApplies(env *resolveEnv) bool {
	if env.syncedUpstream {
		return false
	}
	return liveryResolveNeeded(env.prior, env.upstream.AircraftID, env.upstream.LiveryID)
}

func applyLiveryEnrich(env *resolveEnv) {
	if !env.hasFlight && env.prior != nil {
		env.flight = *env.prior
		env.hasFlight = true
	}
	names, outcome := env.ctx.Deps.Lookup.Resolve(env.upstream.LiveryID, env.upstream.AircraftID)
	env.ctx.Deps.Metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
	switch outcome {
	case gameliveries.MatchLivery, gameliveries.MatchAircraftOnly:
		if names.AircraftName != "" {
			env.flight.AircraftName = names.AircraftName
		}
		if names.LiveryName != "" {
			env.flight.LiveryName = names.LiveryName
		}
		env.recordMutated = true
	case gameliveries.MatchMiss:
	}
}

func liveryResolveNeeded(prior *flights.Flight, aircraftID, liveryID string) bool {
	if prior == nil {
		return true
	}
	if prior.AircraftID != aircraftID || prior.LiveryID != liveryID {
		return true
	}
	return prior.AircraftName == ""
}

func lookupLiveryNames(env *resolveEnv) *gameliveries.ResolvedNames {
	names, outcome := env.ctx.Deps.Lookup.Resolve(env.upstream.LiveryID, env.upstream.AircraftID)
	env.ctx.Deps.Metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
	switch outcome {
	case gameliveries.MatchLivery, gameliveries.MatchAircraftOnly:
		return &names
	case gameliveries.MatchMiss:
	}
	return nil
}
