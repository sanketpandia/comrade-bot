package flightresolution

import (
	"context"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
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
	LiveVACatalog  virtualairlines.LiveVACatalog
	History        flightstate.History
}

// EffectInput is the read-only input passed to each resolution effect.
type EffectInput struct {
	Ctx     ComputeContext
	Raw     flights.Observation
	Prior   *flights.Record
	Facts   Facts
	History flightstate.History
}

// Outcome tells the flights job what cache I/O to perform after resolution.
type Outcome struct {
	UpdateFlightRecord bool
	FPLSyncScheduled   bool
}

// ResolvedFlight is the enriched flight record plus job-oriented outcome flags.
type ResolvedFlight struct {
	Record  flights.Record
	Outcome Outcome
}

// ResolveFlight runs composable effect decisions for one upstream flight tick.
func ResolveFlight(ctx ComputeContext, upstream infiniteflight.Flight, prior *flights.Record) ResolvedFlight {
	raw := flights.NewObservation(upstream, ctx.Session)
	facts := computeFacts(ctx, raw, prior)

	derived := flights.Derived{}
	if prior != nil {
		derived = prior.Derived.Clone()
	}

	outcome := Outcome{}
	in := EffectInput{
		Ctx:     ctx,
		Raw:     raw,
		Prior:   prior,
		Facts:   facts,
		History: ctx.History,
	}

	for _, effect := range flightEffects {
		if effect.Applies(in) {
			effect.Apply(in, &derived, &outcome)
		}
	}

	record := flights.Record{Raw: raw, Derived: derived}
	updateRecord := facts.MotionChanged || facts.IsNew || prior == nil || !derived.Equal(prior.Derived)

	return ResolvedFlight{
		Record: record,
		Outcome: Outcome{
			UpdateFlightRecord: updateRecord,
			FPLSyncScheduled:   outcome.FPLSyncScheduled,
		},
	}
}

func RunFPLSync(_ context.Context, _ flights.Record) {
	// Placeholder until Infinite Flight flight-plan sync is implemented.
}
