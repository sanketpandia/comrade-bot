package flightresolution

import (
	"infinite-experiment/politburo/internal/livegame/flights"
)

// Facts are computed once per tick and shared by all effects.
type Facts struct {
	IsNew          bool
	MotionChanged  bool
	FPLDue         bool
	LiveryChanged  bool
}

func computeFacts(ctx ComputeContext, raw flights.Observation, prior *flights.Record) Facts {
	facts := Facts{IsNew: prior == nil}

	motion := flights.MotionFromObservation(raw)
	if !ctx.HasPriorMotion {
		facts.MotionChanged = true
	} else if !flights.MotionEqual(motion, ctx.PriorMotion) {
		facts.MotionChanged = true
	}

	var priorPathSync flights.PathSync
	if prior != nil {
		priorPathSync = prior.Derived.PathSync
		if prior.Raw.AircraftID != raw.AircraftID || prior.Raw.LiveryID != raw.LiveryID {
			facts.LiveryChanged = true
		}
	} else {
		facts.LiveryChanged = true
	}

	facts.FPLDue = flights.FPLSyncDue(raw.PilotState, priorPathSync, ctx.RefreshedAt)
	return facts
}
