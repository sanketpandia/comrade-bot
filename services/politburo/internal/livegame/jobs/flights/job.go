// Package flights runs the scheduled job that polls Infinite Flight for live
// flights per active session and writes normalized snapshots to Redis.
package flights

import (
	"context"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/scheduler"
)

const jobName = "infinite-flight-flights"

// LiveVACatalogLoader loads VA affix rules for live flight classification.
type LiveVACatalogLoader interface {
	LoadLiveFlightVACatalog(ctx context.Context) (virtualairlines.LiveVACatalog, error)
}

// JobExecutor runs a job with scheduler-level metrics (e.g. chained after-sync work).
type JobExecutor interface {
	Execute(context.Context, scheduler.Job)
}

type Job struct {
	client        infiniteflight.FlightsClient
	cache         cache.Store
	lookup        *gameliveries.Lookup
	vaCatalog     LiveVACatalogLoader
	metrics       *metrics.Registry
	executor      JobExecutor
	afterSync     scheduler.Job
	now           func() time.Time
}

func New(
	client infiniteflight.FlightsClient,
	cacheStore cache.Store,
	lookup *gameliveries.Lookup,
	vaCatalog LiveVACatalogLoader,
	metricsRegistry *metrics.Registry,
	executor JobExecutor,
	afterSync scheduler.Job,
) *Job {
	return &Job{
		client: client, cache: cacheStore, lookup: lookup, vaCatalog: vaCatalog,
		metrics: metricsRegistry, executor: executor, afterSync: afterSync, now: time.Now,
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
	liveVACatalog := virtualairlines.LiveVACatalog{Servers: map[string][]string{}}
	if j.vaCatalog != nil {
		catalog, err := j.vaCatalog.LoadLiveFlightVACatalog(ctx)
		if err != nil {
			slog.Error("failed to load live VA catalog; continuing with empty catalog", "error", err)
		} else {
			liveVACatalog = catalog
		}
	}
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
	if j.executor != nil && j.afterSync != nil {
		j.executor.Execute(ctx, j.afterSync)
	}
	return nil
}
