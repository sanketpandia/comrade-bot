package webhooks

import (
	"database/sql"

	"infinite-experiment/politburo/internal/cache"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
	webhookjobs "infinite-experiment/politburo/internal/webhooks/jobs"
	"infinite-experiment/politburo/internal/webhooks/runlog"
)

// NewJob builds the webhook dispatcher used by both the scheduler (jobs.Register) and
// the manual HTTP trigger, so both paths share handlers, metrics, and run logging.
func NewJob(cacheStore cache.Store, liveryLookup *gameliveries.Lookup, db *sql.DB, metricsRegistry *metrics.Registry) *webhookjobs.Job {
	return webhookjobs.New(
		vaconfig.NewRepository(db),
		NewRegistry(cacheStore, liveryLookup, metricsRegistry),
		runlog.NewRepository(db),
		metricsRegistry,
	)
}
