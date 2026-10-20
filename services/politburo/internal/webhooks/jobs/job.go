// Package jobs runs the scheduled VA webhook dispatcher and on-demand triggers.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/metrics"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	"infinite-experiment/politburo/internal/webhooks/runlog"
)

const JobName = "infinite-flight-webhooks"

// ErrFeatureNotFound means the VA's WEBHOOKS config has no entry for the requested feature.
var ErrFeatureNotFound = errors.New("webhook type not found")

type configLoader interface {
	ListActiveWebhooks(ctx context.Context) ([]vaconfig.ActiveWebhooks, error)
	GetActiveWebhooks(ctx context.Context, vaID string) (vaconfig.ActiveWebhooks, error)
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
	rows, err := j.loader.ListActiveWebhooks(ctx)
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
			due = append(due, dispatchItem(row, entry))
		}
	}

	var mu sync.Mutex
	var insertErr error
	err = dispatch.Run(ctx, dispatch.DefaultWorkers, len(due), func(ctx context.Context, i int) error {
		_, err := j.DispatchOne(ctx, due[i])
		if err != nil {
			slog.Error("record webhook run", "va_id", due[i].VAID, "webhook_type", due[i].WebhookType, "error", err)
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

// Trigger dispatches one feature from an active VA's WEBHOOKS config immediately, regardless
// of its frequency. It returns vaconfig.ErrNotFound when the VA has no WEBHOOKS config and
// ErrFeatureNotFound when the config has no entry for webhookType.
func (j *Job) Trigger(ctx context.Context, vaID, webhookType string) (dispatch.Result, error) {
	webhookType = strings.TrimSpace(webhookType)
	row, err := j.loader.GetActiveWebhooks(ctx, vaID)
	if err != nil {
		return dispatch.Result{}, err
	}
	entries, err := webhookconfig.Parse(row.Webhooks)
	if err != nil {
		return dispatch.Result{}, fmt.Errorf("parse webhooks config: %w", err)
	}
	for _, entry := range entries {
		if entry.Key == webhookType {
			return j.DispatchOne(ctx, dispatchItem(row, entry))
		}
	}
	return dispatch.Result{}, ErrFeatureNotFound
}

// DispatchOne runs one webhook feature, records metrics, and persists a run log row.
func (j *Job) DispatchOne(ctx context.Context, item dispatch.Item) (dispatch.Result, error) {
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
		return result, err
	}
	return result, nil
}

func dispatchItem(row vaconfig.ActiveWebhooks, entry webhookconfig.Entry) dispatch.Item {
	return dispatch.Item{
		VAID:             row.VAID,
		VACode:           row.VACode,
		WebhookType:      entry.Key,
		URL:              entry.URL,
		Frequency:        entry.Frequency,
		Filters:          entry.Filters,
		PreferredServers: row.PreferredServers,
	}
}
