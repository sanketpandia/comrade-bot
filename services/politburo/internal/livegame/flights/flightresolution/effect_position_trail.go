package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var positionTrailEffect = EffectDecision{
	Name:    "PositionTrail",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyPositionTrail,
}

func applyPositionTrail(in EffectInput, analytics *flights.Analytics, _ *flights.Actions, _ *flights.Links, outcome *Outcome) {
	phase := in.History.State.Phase
	activity := in.History.State.Activity
	if fs := analytics.FlightState; fs != nil {
		phase = fs.Phase
		activity = fs.Activity
	}
	trail, changed := flights.AppendPositionFix(outcome.NextPositionTrail, phase, activity, in.Raw, in.Ctx.RefreshedAt)
	if !changed {
		return
	}
	outcome.NextPositionTrail = trail
	outcome.PositionTrailDirty = true
}
