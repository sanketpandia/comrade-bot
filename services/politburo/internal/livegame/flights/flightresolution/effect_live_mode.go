package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var liveModeCheckEffect = EffectDecision{
	Name:    "LiveModeCheck",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyLiveModeCheck,
}

func applyLiveModeCheck(_ EffectInput, _ *flights.Analytics, _ *flights.Actions, _ *flights.Links, _ *Outcome) {
	// Future: match raw.AircraftID against Live organization fleet aircraft.
}
