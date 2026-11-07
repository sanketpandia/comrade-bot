package flightplans

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

const jobName = "infinite-flight-flight-plans"

const maxDuePerRun = 30

type Job struct {
	fetcher *flights.PlanFetcher
	cache   cache.Store
}

func New(client infiniteflight.FlightsClient, cacheStore cache.Store) *Job {
	return &Job{fetcher: flights.NewPlanFetcher(client, cacheStore), cache: cacheStore}
}

func (j *Job) Name() string {
	return jobName
}

func (j *Job) Run(ctx context.Context) error {
	var due []flights.FPLDueEntry
	if err := j.cache.GetJSON(ctx, cache.KeyFPLDueList, &due); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			return nil
		}
		return fmt.Errorf("read fpl due list: %w", err)
	}
	if len(due) == 0 {
		return nil
	}
	if len(due) > maxDuePerRun {
		due = due[:maxDuePerRun]
	}

	synced := 0
	for _, entry := range due {
		if entry.SessionID == "" || entry.FlightID == "" {
			continue
		}
		if _, err := j.fetcher.FetchDue(ctx, entry.SessionID, entry.FlightID); err != nil {
			slog.Warn("background flight plan sync failed", "sessionId", entry.SessionID, "flightId", entry.FlightID, "error", err)
			continue
		}
		synced++
	}
	slog.Info("Infinite Flight flight plans synced", "attempted", len(due), "synced", synced)
	return nil
}
