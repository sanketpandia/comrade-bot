package jobs

import (
	"database/sql"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	flightsjob "infinite-experiment/politburo/internal/livegame/jobs/flights"
	liveriesjob "infinite-experiment/politburo/internal/livegame/jobs/liveries"
	sessionsjob "infinite-experiment/politburo/internal/livegame/jobs/sessions"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/scheduler"
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
	return jobScheduler.Register(flightsjob.New(client, cacheStore, lookup, metricsRegistry), cache.ScheduleFlightsSync)
}
