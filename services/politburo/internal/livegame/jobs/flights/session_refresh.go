package flights

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightresolution"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func (j *Job) refreshSession(ctx context.Context, session infiniteflight.Session, refreshedAt time.Time, liveVACatalog virtualairlines.LiveVACatalog) (int, error) {
	items, err := j.fetchUpstreamFlights(ctx, session)
	if err != nil {
		return 0, err
	}

	existing := j.loadTracksSnapshot(ctx, session.NormalizedName)
	stats := sessionRefreshStats{}
	tracks := make(map[string]gameflights.FlightMotion, len(items))
	pilotStateCounts := map[string]float64{}

	computeBase := flightresolution.ComputeContext{
		Context:       ctx,
		Deps:          flightresolution.ComputeDeps{Lookup: j.lookup, Metrics: j.metrics},
		Session:       session,
		RefreshedAt:   refreshedAt,
		LiveVACatalog: liveVACatalog,
	}

	for _, item := range items {
		prior, cacheMiss, err := j.loadPriorFlightRecord(ctx, item.FlightID)
		if err != nil {
			return 0, err
		}
		if cacheMiss {
			stats.recordCacheMisses++
		}

		computeCtx := computeBase
		computeCtx.PriorMotion, computeCtx.HasPriorMotion = existing.Tracks[item.FlightID]

		resolved := flightresolution.ResolveFlight(computeCtx, item, prior)
		if err := j.applyResolvedFlight(ctx, session, item.FlightID, resolved, &stats); err != nil {
			return 0, err
		}

		tracks[item.FlightID] = gameflights.MotionFromUpstream(item)
		pilotStateCounts[resolved.Flight.Normalized.PilotState]++
	}

	return j.finalizeSession(ctx, session, items, tracks, pilotStateCounts, refreshedAt, stats)
}

func (j *Job) fetchUpstreamFlights(ctx context.Context, session infiniteflight.Session) ([]infiniteflight.Flight, error) {
	upstream, err := j.client.GetSessionFlights(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("fetch flights: %w", err)
	}
	if len(upstream) >= gameflights.MaxFlightsPerRequest {
		slog.Warn("Infinite Flight flights response hit the max record cap", "sessionId", session.ID, "server", session.NormalizedName, "flights", len(upstream))
	}
	return dedupeUpstream(upstream), nil
}

func (j *Job) loadTracksSnapshot(ctx context.Context, normalizedName string) gameflights.Snapshot {
	var existing gameflights.Snapshot
	if err := j.cache.GetJSON(ctx, cache.KeyActiveFlights(normalizedName), &existing); err != nil {
		if err != cache.ErrMiss {
			slog.Warn("failed to read existing flights snapshot; treating as empty", "server", normalizedName, "error", err)
		}
		existing.Tracks = nil
	}
	return existing
}

func (j *Job) loadPriorFlightRecord(ctx context.Context, flightID string) (*gameflights.Flight, bool, error) {
	var priorFlight gameflights.Flight
	if err := j.cache.GetJSON(ctx, cache.KeyFlightRecord(flightID), &priorFlight); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("read flight record %s: %w", flightID, err)
	}
	return &priorFlight, false, nil
}

func (j *Job) applyResolvedFlight(ctx context.Context, session infiniteflight.Session, flightID string, resolved flightresolution.ResolvedFlight, stats *sessionRefreshStats) error {
	server := session.NormalizedName
	if resolved.Outcome.UpdateFlightRecord {
		stats.resolved++
		j.metrics.FlightsComputeTotal.WithLabelValues(server, "resolved").Inc()
	} else {
		stats.skipped++
		j.metrics.FlightsComputeTotal.WithLabelValues(server, "skipped").Inc()
	}

	if resolved.Outcome.FPLSyncScheduled {
		stats.fplScheduled++
		j.metrics.FlightsFPLSyncScheduledTotal.WithLabelValues(server).Inc()
	}

	if !resolved.Outcome.UpdateFlightRecord {
		j.metrics.FlightsRecordUpdateTotal.WithLabelValues(server, "skipped").Inc()
		stats.recordSkipped++
		return nil
	}

	if err := j.cache.SetJSON(ctx, cache.KeyFlightRecord(flightID), resolved.Flight, cache.ActiveFlightsTTL); err != nil {
		return fmt.Errorf("cache flight record %s: %w", flightID, err)
	}
	j.metrics.FlightsRecordUpdateTotal.WithLabelValues(server, "written").Inc()
	stats.recordWritten++
	return nil
}

func (j *Job) finalizeSession(
	ctx context.Context,
	session infiniteflight.Session,
	items []infiniteflight.Flight,
	tracks map[string]gameflights.FlightMotion,
	pilotStateCounts map[string]float64,
	refreshedAt time.Time,
	stats sessionRefreshStats,
) (int, error) {
	server := session.NormalizedName
	snapshot := gameflights.Snapshot{Tracks: tracks, LastCached: refreshedAt}
	if snapshot.Tracks == nil {
		snapshot.Tracks = make(map[string]gameflights.FlightMotion)
	}

	j.metrics.FlightsActive.WithLabelValues(server).Set(float64(len(tracks)))
	for _, state := range gameflights.PilotStateNames() {
		j.metrics.FlightsByPilotState.WithLabelValues(server, state).Set(pilotStateCounts[state])
	}
	if stats.recordCacheMisses > 0 {
		j.metrics.FlightsRecordCacheMissTotal.WithLabelValues(server).Add(float64(stats.recordCacheMisses))
	}

	if err := j.cache.SetJSON(ctx, cache.KeyActiveFlights(server), snapshot, cache.ActiveFlightsTTL); err != nil {
		return 0, fmt.Errorf("cache flights: %w", err)
	}

	slog.Debug("session flights refresh complete",
		"server", server,
		"upstream", len(items),
		"skipped", stats.skipped,
		"resolved", stats.resolved,
		"fplScheduled", stats.fplScheduled,
		"recordWritesSkipped", stats.recordSkipped,
		"recordWrites", stats.recordWritten,
		"recordCacheMisses", stats.recordCacheMisses,
	)
	return len(tracks), nil
}
