package flights

import (
	"context"
	"time"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/metrics"
)

type FlightComputeDecision struct {
	UpdateFlightRecord bool
	FPLSync            bool
	FastPath           bool
}

type FlightComputeResult struct {
	Flight   Flight
	Decision FlightComputeDecision
}

type ComputeDeps struct {
	Lookup  *gameliveries.Lookup
	Metrics *metrics.Registry
}

func ComputeFlight(
	deps ComputeDeps,
	session infiniteflight.Session,
	upstream infiniteflight.Flight,
	prior *Flight,
	priorMotion FlightMotion,
	hasPriorMotion bool,
	refreshedAt time.Time,
) FlightComputeResult {
	var priorPathSync *PathSync
	if prior != nil {
		priorPathSync = prior.PathSync
	}

	fplSync := FPLSyncDue(upstream.PilotState, priorPathSync, refreshedAt)
	motion := MotionFromUpstream(upstream)

	if prior != nil && hasPriorMotion && MotionEqual(motion, priorMotion) && !fplSync {
		return FlightComputeResult{
			Flight: *prior,
			Decision: FlightComputeDecision{
				FPLSync:  false,
				FastPath: true,
			},
		}
	}

	var resolved *gameliveries.ResolvedNames
	if ShouldResolveLivery(prior, upstream.AircraftID, upstream.LiveryID) {
		names, outcome := deps.Lookup.Resolve(upstream.LiveryID, upstream.AircraftID)
		deps.Metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
		switch outcome {
		case gameliveries.MatchLivery, gameliveries.MatchAircraftOnly:
			resolved = &names
		case gameliveries.MatchMiss:
		}
	}

	flight := MapFlight(upstream, session, resolved, prior, refreshedAt)
	pathSync := flight.PathSync
	if pathSync == nil {
		pathSync = &PathSync{}
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

func RunFPLSync(_ context.Context, _ Flight) {
	// Placeholder until Infinite Flight flight-plan sync is implemented.
}
