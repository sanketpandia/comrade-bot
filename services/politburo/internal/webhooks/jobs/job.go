// Package jobs runs the scheduled VA webhook dispatcher.
package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/metrics"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	"infinite-experiment/politburo/internal/webhooks/runlog"
)

const JobName = "infinite-flight-webhooks"

type configLoader interface {
	LoadWebhookDispatchRows(ctx context.Context) ([]virtualairlines.WebhookDispatchRow, error)
}

type runInserter interface {
	Insert(ctx context.Context, row runlog.Row) error
}

// Job loads WEBHOOKS configs and dispatches feature entries that are due this tick.
type Job struct {
	loader   configLoader
	registry *dispatch.Registry
	runs     runInserter
	metrics  *metrics.Registry
	now      func() time.Time
}

func New(loader configLoader, registry *dispatch.Registry, runs runInserter, reg *metrics.Registry) *Job {
	return &Job{
		loader:   loader,
		registry: registry,
		runs:     runs,
		metrics:  reg,
		now:      time.Now,
	}
}

func (j *Job) Name() string {
	return JobName
}

func (j *Job) Run(ctx context.Context) error {
	rows, err := j.loader.LoadWebhookDispatchRows(ctx)
	if err != nil {
		return err
	}
	tick := j.now().UTC()
	var due []dispatch.Item
	for _, row := range rows {
		entries, err := webhookconfig.Parse(row.Webhooks)
		if err != nil {
			slog.Warn("skip VA webhook config", "va_id", row.VAID, "error", err)
			continue
		}
		for _, entry := range entries {
			if !webhookconfig.Due(entry.Frequency, tick) {
				continue
			}
			due = append(due, dispatch.Item{
				VAID:             row.VAID,
				VACode:           row.VACode,
				WebhookType:      entry.Key,
				URL:              entry.URL,
				Filters:          entry.Filters,
				PreferredServers: row.PreferredServers,
			})
		}
	}

	var mu sync.Mutex
	var insertErr error
	err = dispatch.Run(ctx, dispatch.DefaultWorkers, len(due), func(ctx context.Context, i int) error {
		item := due[i]
		started := time.Now().UTC()
		result := j.registry.Handle(ctx, item)
		finished := time.Now().UTC()
		if result.Outcome == "" {
			result.Outcome = dispatch.OutcomeError
			if result.ErrorMessage == "" {
				result.ErrorMessage = "webhook handler returned an empty outcome"
			}
		}
		j.metrics.ObserveWebhookDispatch(item.WebhookType, result.Outcome, finished.Sub(started).Seconds())
		if err := j.runs.Insert(ctx, runlog.Row{
			VAID:            item.VAID,
			WebhookType:     item.WebhookType,
			Outcome:         result.Outcome,
			SkipReason:      result.SkipReason,
			DiscordMessages: result.DiscordMessages,
			ErrorMessage:    result.ErrorMessage,
			StartedAt:       started,
			FinishedAt:      finished,
		}); err != nil {
			slog.Error("record webhook run", "va_id", item.VAID, "webhook_type", item.WebhookType, "error", err)
			mu.Lock()
			insertErr = errors.Join(insertErr, err)
			mu.Unlock()
		}
		return nil
	})
	if err != nil {
		return err
	}
	return insertErr
}
