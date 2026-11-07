package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var fplSyncUpdateEffect = EffectDecision{
	Name:    "FPLSyncUpdate",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyFPLSyncUpdate,
}

func applyFPLSyncUpdate(in EffectInput, _ *flights.Analytics, actions *flights.Actions, links *flights.Links, outcome *Outcome) {
	vaLinked := len(links.VirtualAirlines) > 0
	due := flights.PlanSyncDue(vaLinked, in.Raw.PilotState, in.Ctx.PlanFetchedAt, in.Ctx.RefreshedAt)
	actions.FPLSync.Required = due
	if vaLinked && !in.Ctx.PlanFetchedAt.IsZero() {
		actions.FPLSync.LastAt = in.Ctx.PlanFetchedAt
	}
	outcome.FPLSyncScheduled = due

	if !in.Ctx.PlanLoaded || !flights.PlanFresh(in.Ctx.Plan.FetchedAt, in.Ctx.RefreshedAt) {
		return
	}
	departure, destination, ok := flights.PlanRouteEndpoints(in.Ctx.Plan)
	if !ok {
		return
	}
	next := flights.RecordNormalized{Departure: departure, Destination: destination}
	priorNorm := flights.RecordNormalized{}
	if in.Prior != nil {
		priorNorm = in.Prior.Normalized
	}
	if next != priorNorm {
		outcome.NormalizedEndpoints = next
		outcome.NormalizedDirty = true
	}
}
