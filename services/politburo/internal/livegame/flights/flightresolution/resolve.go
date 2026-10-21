package flightresolution

import (
	"context"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

type ComputeDeps struct {
	Lookup      *gameliveries.Lookup
	Metrics     *metrics.Registry
	FlightState flightstate.Config
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
	NextFlightState    *flightstate.State
}

// PhaseTransition describes a committed phase change for metrics and logs.
type PhaseTransition struct {
	From flightstate.Phase
	To   flightstate.Phase
}

// ResolvedFlight is the enriched flight record plus job-oriented outcome flags.
type ResolvedFlight struct {
	Record      flights.Record
	Outcome     Outcome
	NextHistory flightstate.History
	Transition  *PhaseTransition
}

// ResolveFlight runs composable effect decisions for one upstream flight tick.
func ResolveFlight(ctx ComputeContext, upstream infiniteflight.Flight, prior *flights.Record, history flightstate.History) ResolvedFlight {
	raw := flights.NewObservation(upstream, ctx.Session, ctx.RefreshedAt)
	facts := computeFacts(ctx, raw, prior)

	cfg := ctx.Deps.FlightState
	if cfg.HistorySize == 0 && cfg.HistoryWindow == 0 {
		cfg = flightstate.DefaultConfig()
	}
	pushed, pushReason := history.Push(flights.FlightStateSample(raw), cfg)
	facts.HistoryPush = pushReason

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
		History: pushed,
	}

	for _, effect := range flightEffects {
		if effect.Applies(in) {
			effect.Apply(in, &derived, &outcome)
		}
	}

	record := flights.Record{Raw: raw, Derived: derived}
	updateRecord := facts.MotionChanged || facts.IsNew || !derived.Equal(priorDerived(prior))

	resolved := ResolvedFlight{
		Record: record,
		Outcome: Outcome{
			UpdateFlightRecord: updateRecord,
			FPLSyncScheduled:   outcome.FPLSyncScheduled,
			NextFlightState:    outcome.NextFlightState,
		},
	}

	if outcome.NextFlightState != nil {
		resolved.NextHistory = pushed.WithState(*outcome.NextFlightState)
		prevPhase := pushed.State.Phase
		if prevPhase != outcome.NextFlightState.Phase {
			resolved.Transition = &PhaseTransition{From: prevPhase, To: outcome.NextFlightState.Phase}
		}
	} else {
		resolved.NextHistory = pushed
	}

	return resolved
}

func priorDerived(prior *flights.Record) flights.Derived {
	if prior == nil {
		return flights.Derived{}
	}
	return prior.Derived
}

func RunFPLSync(_ context.Context, _ flights.Record) {
	// Placeholder until Infinite Flight flight-plan sync is implemented.
}

// LogPhaseTransition emits a debug line when the estimator commits a new phase.
func LogPhaseTransition(flightID string, tr PhaseTransition, confidence float64, reasons []string) {
	if tr.From == tr.To {
		return
	}
	slog.Debug("flight phase transition",
		"flightId", flightID,
		"from", tr.From,
		"to", tr.To,
		"confidence", confidence,
		"reasons", reasons,
	)
}
