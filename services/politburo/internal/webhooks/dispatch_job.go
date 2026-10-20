package webhooks

import (
	"database/sql"

	"infinite-experiment/politburo/internal/cache"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/metrics"
	webhookjobs "infinite-experiment/politburo/internal/webhooks/jobs"
	"infinite-experiment/politburo/internal/webhooks/runlog"
)

// NewJob builds the webhook dispatcher used by both the scheduler (jobs.Register) and
// the manual HTTP trigger, so both paths share handlers, metrics, and run logging.
func NewJob(cacheStore cache.Store, db *sql.DB, metricsRegistry *metrics.Registry) *webhookjobs.Job {
	return webhookjobs.New(
		vaconfig.NewRepository(db),
		NewRegistry(cacheStore, metricsRegistry),
		runlog.NewRepository(db),
		metricsRegistry,
	)
}
