// Package flights runs the scheduled job that polls Infinite Flight for live
// flights per active session and writes normalized snapshots to Redis.
package flights

import (
	"context"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
)

const jobName = "infinite-flight-flights"

// BasicConfigLister loads active VAs' BASIC config for live flight classification.
type BasicConfigLister interface {
	ListActiveBasic(ctx context.Context) ([]vaconfig.ActiveBasic, error)
}

type Job struct {
	client    infiniteflight.FlightsClient
	cache     cache.Store
	lookup    *gameliveries.Lookup
	vaConfigs BasicConfigLister
	metrics   *metrics.Registry
	now       func() time.Time
}

func New(
	client infiniteflight.FlightsClient,
	cacheStore cache.Store,
	lookup *gameliveries.Lookup,
	vaConfigs BasicConfigLister,
	metricsRegistry *metrics.Registry,
) *Job {
	return &Job{
		client: client, cache: cacheStore, lookup: lookup, vaConfigs: vaConfigs,
		metrics: metricsRegistry, now: time.Now,
	}
}

func (j *Job) Name() string {
	return jobName
}

type sessionRefreshStats struct {
	skipped           int
	resolved          int
	fplScheduled      int
	recordWritten     int
	recordSkipped     int
	recordCacheMisses int
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
	liveVACatalog := j.loadLiveVACatalog(ctx)
	totalFlights := 0
	for _, session := range sessionsSnapshot.Result {
		if session.ID == "" || session.NormalizedName == "" {
			continue
		}
		count, err := j.refreshSession(ctx, session, refreshedAt, liveVACatalog)
		if err != nil {
			slog.Error("failed to refresh session flights", "sessionId", session.ID, "server", session.NormalizedName, "error", err)
			continue
		}
		totalFlights += count
	}
	slog.Info("Infinite Flight flights refreshed",
		"sessions", len(sessionsSnapshot.Result),
		"flights", totalFlights,
		"vaCatalogRules", len(liveVACatalog.Rules),
	)
	return nil
}

// loadLiveVACatalog never fails the run: without a catalog, flights still refresh
// and simply carry no VA matches.
func (j *Job) loadLiveVACatalog(ctx context.Context) virtualairlines.LiveVACatalog {
	if j.vaConfigs == nil {
		return virtualairlines.BuildLiveVACatalog(nil)
	}
	rows, err := j.vaConfigs.ListActiveBasic(ctx)
	if err != nil {
		slog.Error("failed to load live VA catalog; continuing with empty catalog", "error", err)
		return virtualairlines.BuildLiveVACatalog(nil)
	}
	return virtualairlines.BuildLiveVACatalog(rows)
}
