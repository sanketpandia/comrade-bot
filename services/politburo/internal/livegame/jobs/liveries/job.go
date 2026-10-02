package liveries

import (
	"context"
	"fmt"
	"log/slog"

	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

const jobName = "infinite-flight-liveries"

type Job struct {
	client infiniteflight.LiveriesClient
	repo   *gameliveries.Repository
	lookup *gameliveries.Lookup
}

func New(client infiniteflight.LiveriesClient, repo *gameliveries.Repository, lookup *gameliveries.Lookup) *Job {
	return &Job{client: client, repo: repo, lookup: lookup}
}

func (j *Job) Name() string {
	return jobName
}

func (j *Job) Run(ctx context.Context) error {
	upstream, err := j.client.GetAircraftLiveries(ctx)
	if err != nil {
		return fmt.Errorf("refresh liveries: %w", err)
	}

	count, err := j.repo.UpsertFromUpstream(ctx, upstream)
	if err != nil {
		return fmt.Errorf("persist liveries: %w", err)
	}
	if err := j.lookup.Reload(ctx); err != nil {
		return fmt.Errorf("reload livery lookup: %w", err)
	}
	slog.Info("Infinite Flight liveries refreshed", "liveries", count)
	return nil
}
