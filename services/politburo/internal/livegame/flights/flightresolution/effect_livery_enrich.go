package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var liveryResolveMetricEffect = EffectDecision{
	Name:    "LiveryResolveMetric",
	Applies: liveryResolveMetricApplies,
	Apply:   applyLiveryResolveMetric,
}

func liveryResolveMetricApplies(in EffectInput) bool {
	return in.Facts.IsNew || in.Facts.LiveryChanged
}

func applyLiveryResolveMetric(in EffectInput, _ *flights.Analytics, _ *flights.Actions, _ *flights.Links, _ *Outcome) {
	_, outcome := in.Ctx.Deps.Lookup.Resolve(in.Raw.LiveryID, in.Raw.AircraftID)
	in.Ctx.Deps.Metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
}
