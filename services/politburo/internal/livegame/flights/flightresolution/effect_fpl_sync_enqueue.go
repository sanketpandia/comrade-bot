package flightresolution

import "infinite-experiment/politburo/internal/livegame/flights"

var fplSyncEnqueueEffect = EffectDecision{
	Name:    "FPLSyncEnqueue",
	Applies: func(in EffectInput) bool { return in.Facts.FPLDue },
	Apply:   applyFPLSyncEnqueue,
}

func applyFPLSyncEnqueue(_ EffectInput, _ *flights.Derived, outcome *Outcome) {
	outcome.FPLSyncScheduled = true
}
