package flightresolution

import (
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

var flightStateEffect = EffectDecision{
	Name:    "FlightState",
	Applies: func(EffectInput) bool { return true },
	Apply:   applyFlightState,
}

func applyFlightState(in EffectInput, out *flights.Derived, outcome *Outcome) {
	cfg := in.Ctx.Deps.FlightState
	if cfg.HistorySize == 0 && cfg.HistoryWindow == 0 {
		cfg = flightstate.DefaultConfig()
	}
	next := flightstate.Classify(in.History.State, in.History, cfg, in.Raw.ObservedAt)
	summary := flightstate.SummaryFromState(next, false)
	out.FlightState = &summary
	outcome.NextFlightState = &next
}
