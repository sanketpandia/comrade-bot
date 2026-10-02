package flights

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/metrics"
)

const jobName = "infinite-flight-flights"

type Job struct {
	client  infiniteflight.FlightsClient
	cache   cache.Store
	lookup  *gameliveries.Lookup
	metrics *metrics.Registry
	now     func() time.Time
}

func New(client infiniteflight.FlightsClient, cacheStore cache.Store, lookup *gameliveries.Lookup, metricsRegistry *metrics.Registry) *Job {
	return &Job{client: client, cache: cacheStore, lookup: lookup, metrics: metricsRegistry, now: time.Now}
}

func (j *Job) Name() string {
	return jobName
}

func (j *Job) Run(ctx context.Context) error {
	var sessionsSnapshot gamesessions.Snapshot
	if err := j.cache.GetJSON(ctx, cache.KeyActiveSessions, &sessionsSnapshot); err != nil {
		slog.Warn("skipping flights refresh; active sessions cache unavailable", "error", err)
		return nil
	}
	if len(sessionsSnapshot.Result) == 0 {
		slog.Warn("skipping flights refresh; no active sessions cached")
		return nil
	}

	refreshedAt := j.now().UTC()
	totalFlights := 0
	for _, session := range sessionsSnapshot.Result {
		if session.ID == "" || session.NormalizedName == "" {
			continue
		}
		count, err := j.refreshSession(ctx, session, refreshedAt)
		if err != nil {
			slog.Error("failed to refresh session flights", "sessionId", session.ID, "server", session.NormalizedName, "error", err)
			continue
		}
		totalFlights += count
	}
	slog.Info("Infinite Flight flights refreshed", "sessions", len(sessionsSnapshot.Result), "flights", totalFlights)
	return nil
}

func (j *Job) refreshSession(ctx context.Context, session infiniteflight.Session, refreshedAt time.Time) (int, error) {
	upstream, err := j.client.GetSessionFlights(ctx, session.ID)
	if err != nil {
		return 0, fmt.Errorf("fetch flights: %w", err)
	}
	if len(upstream) >= gameflights.MaxFlightsPerRequest {
		slog.Warn("Infinite Flight flights response hit the max record cap", "sessionId", session.ID, "server", session.NormalizedName, "flights", len(upstream))
	}

	var existing gameflights.Snapshot
	if err := j.cache.GetJSON(ctx, cache.KeyActiveFlights(session.NormalizedName), &existing); err != nil {
		if err != cache.ErrMiss {
			slog.Warn("failed to read existing flights snapshot; treating as empty", "server", session.NormalizedName, "error", err)
		}
		existing.Result = nil
	}

	existingByID := make(map[string]*gameflights.Flight, len(existing.Result))
	for i := range existing.Result {
		existingByID[existing.Result[i].FlightID] = &existing.Result[i]
	}

	mapped := make([]gameflights.Flight, 0, len(upstream))
	pilotStateCounts := map[string]float64{}
	for _, item := range upstream {
		names, outcome := j.lookup.Resolve(item.LiveryID, item.AircraftID)
		j.metrics.LiveryResolveTotal.WithLabelValues(string(outcome)).Inc()
		var resolved *gameliveries.ResolvedNames
		if outcome != gameliveries.MatchMiss {
			resolved = &names
		}
		flight := gameflights.MapFlight(item, session, resolved, existingByID[item.FlightID], refreshedAt)
		mapped = append(mapped, flight)
		pilotStateCounts[flight.Normalized.PilotState]++
	}

	snapshot := gameflights.Snapshot{
		Result:     gameflights.UpsertFlights(existing.Result, mapped),
		LastCached: refreshedAt,
	}
	if snapshot.Result == nil {
		snapshot.Result = make([]gameflights.Flight, 0)
	}
	j.metrics.FlightsActive.WithLabelValues(session.NormalizedName).Set(float64(len(snapshot.Result)))
	for state, count := range pilotStateCounts {
		j.metrics.FlightsByPilotState.WithLabelValues(session.NormalizedName, state).Set(count)
	}

	if err := j.cache.SetJSON(ctx, cache.KeyActiveFlights(session.NormalizedName), snapshot, gameflights.GameActiveFlightTTL); err != nil {
		return 0, fmt.Errorf("cache flights: %w", err)
	}
	return len(snapshot.Result), nil
}
