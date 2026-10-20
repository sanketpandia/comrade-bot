package webhooks

import (
	"database/sql"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/metrics"
	webhookjobs "infinite-experiment/politburo/internal/webhooks/jobs"
	"infinite-experiment/politburo/internal/webhooks/runlog"
)

// NewDispatchJob builds a webhook dispatcher for on-demand runs (for example HTTP triggers).
// It is not registered on the scheduler; scheduled work uses the job created in jobs.Register.
func NewDispatchJob(
	cacheStore cache.Store,
	db *sql.DB,
	vaService *virtualairlines.Service,
	vaRepo *virtualairlines.Repository,
	metricsRegistry *metrics.Registry,
) *webhookjobs.Job {
	return webhookjobs.New(
		vaService,
		NewRegistry(cacheStore, vaRepo, metricsRegistry),
		runlog.NewRepository(db),
		metricsRegistry,
	)
}
