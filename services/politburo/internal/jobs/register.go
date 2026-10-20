package jobs

import (
	"database/sql"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	flightsjob "infinite-experiment/politburo/internal/livegame/jobs/flights"
	liveriesjob "infinite-experiment/politburo/internal/livegame/jobs/liveries"
	sessionsjob "infinite-experiment/politburo/internal/livegame/jobs/sessions"
	vaflightindexjob "infinite-experiment/politburo/internal/livegame/jobs/vaflightindex"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/scheduler"
	"infinite-experiment/politburo/internal/webhooks"
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
	flightsJob := flightsjob.New(client, cacheStore, lookup, vaconfig.NewRepository(db), metricsRegistry)
	if err := jobScheduler.Register(flightsJob, cache.ScheduleFlightsSync); err != nil {
		return err
	}
	indexJob := vaflightindexjob.New(cacheStore, virtualairlines.NewRepository(db), metricsRegistry)
	if err := jobScheduler.Register(indexJob, cache.ScheduleVAFlightIndexSync); err != nil {
		return err
	}
	return jobScheduler.Register(webhooks.NewJob(cacheStore, db, metricsRegistry), cache.ScheduleWebhooksSync)
}
