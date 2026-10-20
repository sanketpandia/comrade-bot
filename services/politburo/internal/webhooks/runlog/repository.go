// Package runlog persists one audit row per webhook feature dispatch.
package runlog

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"unicode/utf8"
)

const insertWebhookRun = `
INSERT INTO public.va_webhook_runs (
    va_id, webhook_type, outcome, skip_reason, discord_messages, error_message, started_at, finished_at, duration_ms
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const (
	maxSkipReason = 100
	maxErrorRunes = 500
)

// Row is one dispatched feature attempt. Webhook URLs are not stored.
type Row struct {
	VAID            string
	WebhookType     string
	Outcome         string
	SkipReason      string
	DiscordMessages int
	ErrorMessage    string
	StartedAt       time.Time
	FinishedAt      time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Insert(ctx context.Context, row Row) error {
	var skip any
	if row.SkipReason != "" {
		skip = truncate(row.SkipReason, maxSkipReason)
	}
	var errMessage any
	if row.ErrorMessage != "" {
		errMessage = truncate(row.ErrorMessage, maxErrorRunes)
	}
	duration := int(row.FinishedAt.Sub(row.StartedAt).Milliseconds())
	if duration < 0 {
		duration = 0
	}
	_, err := r.db.ExecContext(ctx, insertWebhookRun,
		row.VAID,
		row.WebhookType,
		row.Outcome,
		skip,
		row.DiscordMessages,
		errMessage,
		row.StartedAt.UTC(),
		row.FinishedAt.UTC(),
		duration,
	)
	if err != nil {
		return fmt.Errorf("insert va webhook run: %w", err)
	}
	return nil
}

func truncate(value string, maxRunes int) string {
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes])
}
