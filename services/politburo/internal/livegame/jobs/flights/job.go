// Package flights runs the scheduled job that polls Infinite Flight for live
// flights per active session and writes normalized snapshots to Redis.
package flights

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
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

// Run refreshes cached flight snapshots for every session listed in the active
// sessions snapshot. Missing session cache is non-fatal; per-session upstream
// failures are logged and skipped so other servers still refresh.
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

type sessionRefreshStats struct {
	fastPath          int
	fullCompute       int
	fplScheduled      int
	recordWritten     int
	recordSkipped     int
	recordCacheMisses int
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
		existing.Tracks = nil
	}

	items := dedupeUpstream(upstream)
	tracks := make(map[string]gameflights.FlightMotion, len(items))
	pilotStateCounts := map[string]float64{}
	stats := sessionRefreshStats{}

	for _, item := range items {
		var prior *gameflights.Flight
		var priorFlight gameflights.Flight
		if err := j.cache.GetJSON(ctx, cache.KeyFlightRecord(item.FlightID), &priorFlight); err != nil {
			if !errors.Is(err, cache.ErrMiss) {
				return 0, fmt.Errorf("read flight record %s: %w", item.FlightID, err)
			}
			stats.recordCacheMisses++
		} else {
			prior = &priorFlight
		}

		priorMotion, hasPriorMotion := existing.Tracks[item.FlightID]
		out := j.computeFlight(ctx, session, item, prior, priorMotion, hasPriorMotion, refreshedAt)

		if out.Decision.FastPath {
			stats.fastPath++
			j.metrics.FlightsComputeTotal.WithLabelValues(session.NormalizedName, "fast_path").Inc()
		} else {
			stats.fullCompute++
			j.metrics.FlightsComputeTotal.WithLabelValues(session.NormalizedName, "full_compute").Inc()
		}
		if out.Decision.FPLSync {
			stats.fplScheduled++
			j.metrics.FlightsFPLSyncScheduledTotal.WithLabelValues(session.NormalizedName).Inc()
			runFPLSync(ctx, out.Flight)
		}

		switch {
		case out.Decision.FastPath:
			j.metrics.FlightsRecordUpdateTotal.WithLabelValues(session.NormalizedName, "skipped_fast_path").Inc()
			stats.recordSkipped++
		case !out.Decision.UpdateFlightRecord:
			j.metrics.FlightsRecordUpdateTotal.WithLabelValues(session.NormalizedName, "skipped_fast_path").Inc()
			stats.recordSkipped++
		case !EnableFlightRecordWrites:
			j.metrics.FlightsRecordUpdateTotal.WithLabelValues(session.NormalizedName, "skipped_disabled").Inc()
			stats.recordSkipped++
		default:
			if err := j.cache.SetJSON(ctx, cache.KeyFlightRecord(item.FlightID), out.Flight, gameflights.GameActiveFlightTTL); err != nil {
				return 0, fmt.Errorf("cache flight record %s: %w", item.FlightID, err)
			}
			j.metrics.FlightsRecordUpdateTotal.WithLabelValues(session.NormalizedName, "written").Inc()
			stats.recordWritten++
		}

		tracks[item.FlightID] = gameflights.MotionFromUpstream(item)
		pilotStateCounts[out.Flight.Normalized.PilotState]++
	}

	snapshot := gameflights.Snapshot{
		Tracks:     tracks,
		LastCached: refreshedAt,
	}
	if snapshot.Tracks == nil {
		snapshot.Tracks = make(map[string]gameflights.FlightMotion)
	}

	j.metrics.FlightsActive.WithLabelValues(session.NormalizedName).Set(float64(len(tracks)))
	for _, state := range gameflights.PilotStateNames() {
		j.metrics.FlightsByPilotState.WithLabelValues(session.NormalizedName, state).Set(pilotStateCounts[state])
	}
	if stats.recordCacheMisses > 0 {
		j.metrics.FlightsRecordCacheMissTotal.WithLabelValues(session.NormalizedName).Add(float64(stats.recordCacheMisses))
	}

	if err := j.cache.SetJSON(ctx, cache.KeyActiveFlights(session.NormalizedName), snapshot, gameflights.GameActiveFlightTTL); err != nil {
		return 0, fmt.Errorf("cache flights: %w", err)
	}

	slog.Debug("session flights refresh complete",
		"server", session.NormalizedName,
		"upstream", len(items),
		"fastPath", stats.fastPath,
		"fullCompute", stats.fullCompute,
		"fplScheduled", stats.fplScheduled,
		"recordWritesSkipped", stats.recordSkipped,
		"recordWrites", stats.recordWritten,
		"recordCacheMisses", stats.recordCacheMisses,
	)
	return len(tracks), nil
}
