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

func liveryEnrichApplies(in EffectInput) bool {
	if in.Facts.IsNew || in.Facts.LiveryChanged {
		return true
	}
	return in.Prior == nil || in.Prior.Derived.AircraftName == ""
}

func applyLiveryEnrich(in EffectInput, out *flights.Derived, _ *Outcome) {
	names, outcome := in.Ctx.Deps.Lookup.Resolve(in.Raw.LiveryID, in.Raw.AircraftID)
	in.Ctx.Deps.Metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
	switch outcome {
	case gameliveries.MatchLivery, gameliveries.MatchAircraftOnly:
		if names.AircraftName != "" {
			out.AircraftName = names.AircraftName
		}
		if names.LiveryName != "" {
			out.LiveryName = names.LiveryName
		}
	case gameliveries.MatchMiss:
	}
}
