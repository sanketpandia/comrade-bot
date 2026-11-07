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
	PlanFetchedAt  time.Time
	Plan           flights.CachedFlightPlan
	PlanLoaded     bool
	PositionTrail  flights.PositionTrail
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
	NextPositionTrail  flights.PositionTrail
	PositionTrailDirty   bool
	NormalizedDirty      bool
	NormalizedEndpoints  flights.RecordNormalized
}

// PhaseTransition describes a committed phase change for metrics and logs.
type PhaseTransition struct {
	From flightstate.Phase
	To   flightstate.Phase
}

// ResolvedFlight is the enriched flight record plus job-oriented outcome flags.
type ResolvedFlight struct {
	Record             flights.Record
	Outcome            Outcome
	NextHistory        flightstate.History
	NextPositionTrail  flights.PositionTrail
	PositionTrailDirty bool
	Transition         *PhaseTransition
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

	analytics := flights.Analytics{}
	actions := flights.Actions{}
	links := flights.Links{}
	normalized := flights.RecordNormalized{}
	if prior != nil {
		analytics = prior.Analytics.Clone()
		actions = prior.Actions.Clone()
		links = prior.Links.Clone()
		normalized = prior.Normalized
	}

	outcome := Outcome{NextPositionTrail: ctx.PositionTrail}
	in := EffectInput{
		Ctx:     ctx,
		Raw:     raw,
		Prior:   prior,
		Facts:   facts,
		History: pushed,
	}

	for _, effect := range flightEffects {
		if effect.Applies(in) {
			effect.Apply(in, &analytics, &actions, &links, &outcome)
		}
	}

	if outcome.NormalizedDirty {
		normalized = outcome.NormalizedEndpoints
	}

	var nextHistory flightstate.History
	if outcome.NextFlightState != nil {
		nextHistory = pushed.WithState(*outcome.NextFlightState)
	} else {
		nextHistory = pushed
	}

	record := flights.Record{
		Raw:         raw,
		Analytics:   analytics,
		Actions:     actions,
		Links:       links,
		History:     nextHistory,
		Normalized:  normalized,
	}

	var priorRecord flights.Record
	if prior != nil {
		priorRecord = *prior
	}
	updateRecord := facts.MotionChanged || facts.IsNew || !record.StoredSectionsEqual(priorRecord)

	resolved := ResolvedFlight{
		Record: record,
		Outcome: Outcome{
			UpdateFlightRecord: updateRecord,
			FPLSyncScheduled:   outcome.FPLSyncScheduled,
			NextFlightState:    outcome.NextFlightState,
			NextPositionTrail:  outcome.NextPositionTrail,
			PositionTrailDirty: outcome.PositionTrailDirty,
		},
		NextHistory:        nextHistory,
		NextPositionTrail:  outcome.NextPositionTrail,
		PositionTrailDirty: outcome.PositionTrailDirty,
	}

	if outcome.NextFlightState != nil {
		prevPhase := pushed.State.Phase
		if prevPhase != outcome.NextFlightState.Phase {
			resolved.Transition = &PhaseTransition{From: prevPhase, To: outcome.NextFlightState.Phase}
		}
	}

	return resolved
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
