package flightresolution

import (
	"infinite-experiment/politburo/internal/livegame/flights"
)

// Facts are computed once per tick and shared by all effects.
type Facts struct {
	IsNew         bool
	MotionChanged bool
	HistoryPush   string
}

func computeFacts(ctx ComputeContext, raw flights.Observation, prior *flights.Record) Facts {
	facts := Facts{IsNew: prior == nil}

	motion := flights.MotionFromObservation(raw)
	if !ctx.HasPriorMotion {
		facts.MotionChanged = true
	} else if !flights.MotionEqual(motion, ctx.PriorMotion) {
		facts.MotionChanged = true
	}

	return facts
}
