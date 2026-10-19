package flightresolution

import (
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type resolveEnv struct {
	ctx            ComputeContext
	upstream       infiniteflight.Flight
	prior          *flights.Flight
	motion         flights.FlightMotion
	fplDue         bool
	flight         flights.Flight
	hasFlight      bool
	syncedUpstream bool
	recordMutated  bool
	outcome        Outcome
}

func newResolveEnv(ctx ComputeContext, upstream infiniteflight.Flight, prior *flights.Flight) *resolveEnv {
	var priorPathSync *flights.PathSync
	if prior != nil {
		priorPathSync = prior.PathSync
	}
	env := &resolveEnv{
		ctx:      ctx,
		upstream: upstream,
		prior:    prior,
		motion:   flights.MotionFromUpstream(upstream),
		fplDue:   flights.FPLSyncDue(upstream.PilotState, priorPathSync, ctx.RefreshedAt),
	}
	if prior != nil {
		env.flight = *prior
		env.hasFlight = true
	}
	return env
}
