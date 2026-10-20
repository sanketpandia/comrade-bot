package jobs

import (
	"database/sql"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	flightsjob "infinite-experiment/politburo/internal/livegame/jobs/flights"
	liveriesjob "infinite-experiment/politburo/internal/livegame/jobs/liveries"
	sessionsjob "infinite-experiment/politburo/internal/livegame/jobs/sessions"
	vaflightindexjob "infinite-experiment/politburo/internal/livegame/jobs/vaflightindex"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/scheduler"
	"infinite-experiment/politburo/internal/webhooks/discord"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	"infinite-experiment/politburo/internal/webhooks/features/activeflights"
	webhookjobs "infinite-experiment/politburo/internal/webhooks/jobs"
	"infinite-experiment/politburo/internal/webhooks/runlog"
)

// Register is the single composition point for scheduled work.
func Register(
	jobScheduler *scheduler.Scheduler,
	client infiniteflight.ClientAPI,
	cacheStore cache.Store,
	db *sql.DB,
	lookup *gameliveries.Lookup,
	metricsRegistry *metrics.Registry,
) error {
	liveryRepo := gameliveries.NewRepository(db)
	if err := jobScheduler.Register(sessionsjob.New(client, cacheStore), cache.ScheduleSessionsSync); err != nil {
		return err
	}
	if err := jobScheduler.Register(liveriesjob.New(client, liveryRepo, lookup), cache.ScheduleLiveriesSync); err != nil {
		return err
	}
	vaRepo := virtualairlines.NewRepository(db)
	vaService := virtualairlines.NewService(db, users.NewRepository(db), vaRepo)
	indexJob := vaflightindexjob.New(cacheStore, metricsRegistry)
	flightsJob := flightsjob.New(client, cacheStore, lookup, vaService, metricsRegistry, jobScheduler, indexJob)
	if err := jobScheduler.Register(flightsJob, cache.ScheduleFlightsSync); err != nil {
		return err
	}

	registry := dispatch.NewRegistry()
	registry.Register(activeflights.FeatureKey, activeflights.New(
		gameflights.NewReader(cacheStore),
		gamesessions.NewReader(cacheStore),
		vaRepo,
		discord.NewClient(metricsRegistry),
		metricsRegistry,
	))
	return jobScheduler.Register(
		webhookjobs.New(vaService, registry, runlog.NewRepository(db), metricsRegistry),
		cache.ScheduleWebhooksSync,
	)
}
