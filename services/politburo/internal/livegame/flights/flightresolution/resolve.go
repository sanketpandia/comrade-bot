package flightresolution

import (
	"context"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

type ComputeDeps struct {
	Lookup  *gameliveries.Lookup
	Metrics *metrics.Registry
}

// ComputeContext carries session tick metadata and injectable deps for ResolveFlight.
type ComputeContext struct {
	Context        context.Context
	Deps           ComputeDeps
	Session        infiniteflight.Session
	RefreshedAt    time.Time
	PriorMotion    flights.FlightMotion
	HasPriorMotion bool
}

// Outcome tells the flights job what cache I/O to perform after resolution.
type Outcome struct {
	UpdateFlightRecord bool
	FPLSyncScheduled   bool
}

// ResolvedFlight is the enriched flight record plus job-oriented outcome flags.
type ResolvedFlight struct {
	Flight  flights.Flight
	Outcome Outcome
}

// ResolveFlight runs composable effect decisions for one upstream flight tick.
func ResolveFlight(ctx ComputeContext, upstream infiniteflight.Flight, prior *flights.Flight) ResolvedFlight {
	env := newResolveEnv(ctx, upstream, prior)
	for _, effect := range flightEffects {
		if effect.Applies(env) {
			effect.Apply(env)
		}
	}

	flight := env.flight
	if prior != nil && !env.recordMutated {
		flight = *prior
	}

	return ResolvedFlight{
		Flight: flight,
		Outcome: Outcome{
			UpdateFlightRecord: env.recordMutated,
			FPLSyncScheduled:   env.outcome.FPLSyncScheduled,
		},
	}
}

func RunFPLSync(_ context.Context, _ flights.Flight) {
	// Placeholder until Infinite Flight flight-plan sync is implemented.
}
