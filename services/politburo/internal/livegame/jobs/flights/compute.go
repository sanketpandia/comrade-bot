package flights

import (
	"context"
	"time"

	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
)

type FlightComputeDecision struct {
	UpdateFlightRecord bool
	FPLSync            bool
	FastPath           bool
}

type FlightComputeResult struct {
	Flight   gameflights.Flight
	Decision FlightComputeDecision
}

func (j *Job) computeFlight(
	_ context.Context,
	session infiniteflight.Session,
	upstream infiniteflight.Flight,
	prior *gameflights.Flight,
	priorMotion gameflights.FlightMotion,
	hasPriorMotion bool,
	refreshedAt time.Time,
) FlightComputeResult {
	var priorPathSync *gameflights.PathSync
	if prior != nil {
		priorPathSync = prior.PathSync
	}

	fplSync := gameflights.FPLSyncDue(upstream.PilotState, priorPathSync, refreshedAt)
	motion := gameflights.MotionFromUpstream(upstream)

	if prior != nil && hasPriorMotion && gameflights.MotionEqual(motion, priorMotion) && !fplSync {
		return FlightComputeResult{
			Flight: *prior,
			Decision: FlightComputeDecision{
				FPLSync:  false,
				FastPath: true,
			},
		}
	}

	var resolved *gameliveries.ResolvedNames
	if gameflights.ShouldResolveLivery(prior, upstream.AircraftID, upstream.LiveryID) {
		names, outcome := j.lookup.Resolve(upstream.LiveryID, upstream.AircraftID)
		j.metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
		switch outcome {
		case gameliveries.MatchLivery, gameliveries.MatchAircraftOnly:
			resolved = &names
		case gameliveries.MatchMiss:
		}
	}

	flight := gameflights.MapFlight(upstream, session, resolved, prior, refreshedAt)
	pathSync := flight.PathSync
	if pathSync == nil {
		pathSync = &gameflights.PathSync{}
		flight.PathSync = pathSync
	}
	pathSync.FPLSyncRequired = fplSync
	if fplSync {
		pathSync.LastFPLSyncAt = refreshedAt
	} else if prior != nil && prior.PathSync != nil && !prior.PathSync.LastFPLSyncAt.IsZero() {
		pathSync.LastFPLSyncAt = prior.PathSync.LastFPLSyncAt
	}

	return FlightComputeResult{
		Flight: flight,
		Decision: FlightComputeDecision{
			UpdateFlightRecord: true,
			FPLSync:            fplSync,
		},
	}
}

func runFPLSync(_ context.Context, _ gameflights.Flight) {
	// Placeholder until Infinite Flight flight-plan sync is implemented.
}
