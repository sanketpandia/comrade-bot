package flights

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

// PlanFetcher loads cached flight plans and fetches from Infinite Flight when stale.
type PlanFetcher struct {
	client infiniteflight.FlightsClient
	cache  cache.Store
	mu     sync.Mutex
	now    func() time.Time
}

func NewPlanFetcher(client infiniteflight.FlightsClient, cacheStore cache.Store) *PlanFetcher {
	return &PlanFetcher{client: client, cache: cacheStore, now: time.Now}
}

func (p *PlanFetcher) LoadCached(ctx context.Context, flightID string) (CachedFlightPlan, bool, error) {
	var plan CachedFlightPlan
	err := p.cache.GetJSON(ctx, cache.KeyFlightPlan(flightID), &plan)
	if errors.Is(err, cache.ErrMiss) {
		return CachedFlightPlan{}, false, nil
	}
	if err != nil {
		return CachedFlightPlan{}, false, fmt.Errorf("read flight plan %s: %w", flightID, err)
	}
	return plan, true, nil
}

// GetOrFetch returns a fresh cached plan or performs one upstream fetch under the shared mutex.
func (p *PlanFetcher) GetOrFetch(ctx context.Context, sessionID, flightID string) (CachedFlightPlan, error) {
	now := p.now().UTC()
	cached, ok, err := p.LoadCached(ctx, flightID)
	if err != nil {
		return CachedFlightPlan{}, err
	}
	if ok && PlanFresh(cached.FetchedAt, now) {
		return cached, nil
	}
	return p.fetchAndStore(ctx, sessionID, flightID)
}

func (p *PlanFetcher) FetchDue(ctx context.Context, sessionID, flightID string) (CachedFlightPlan, error) {
	now := p.now().UTC()
	cached, ok, err := p.LoadCached(ctx, flightID)
	if err != nil {
		return CachedFlightPlan{}, err
	}
	if ok && PlanFresh(cached.FetchedAt, now) {
		return cached, nil
	}
	return p.fetchAndStore(ctx, sessionID, flightID)
}

func (p *PlanFetcher) fetchAndStore(ctx context.Context, sessionID, flightID string) (CachedFlightPlan, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.now().UTC()
	cached, ok, err := p.LoadCached(ctx, flightID)
	if err != nil {
		return CachedFlightPlan{}, err
	}
	if ok && PlanFresh(cached.FetchedAt, now) {
		return cached, nil
	}

	upstream, err := p.client.GetFlightPlan(ctx, sessionID, flightID)
	if err != nil {
		if isDefinitiveNoPlan(err) {
			stored := CachedFlightPlan{FetchedAt: now, Absent: true}
			if setErr := p.cache.SetJSON(ctx, cache.KeyFlightPlan(flightID), stored, cache.ActiveFlightsTTL); setErr != nil {
				return CachedFlightPlan{}, setErr
			}
			return stored, nil
		}
		return CachedFlightPlan{}, err
	}

	stored := CachedFlightPlan{
		FetchedAt:      now,
		FlightPlanID:   upstream.FlightPlanID,
		FlightPlanType: upstream.FlightPlanType,
		Waypoints:      FlattenFlightPlanWaypoints(upstream),
	}
	if err := p.cache.SetJSON(ctx, cache.KeyFlightPlan(flightID), stored, cache.ActiveFlightsTTL); err != nil {
		return CachedFlightPlan{}, fmt.Errorf("cache flight plan %s: %w", flightID, err)
	}
	return stored, nil
}

func isDefinitiveNoPlan(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "errorCode")
}
