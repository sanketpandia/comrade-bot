package valive

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const (
	ActionOutcomeSuccess = "success"
	ActionOutcomeError   = "error"
)

type ActionHistoryRepository struct {
	db *sql.DB
}

func NewActionHistoryRepository(db *sql.DB) *ActionHistoryRepository {
	return &ActionHistoryRepository{db: db}
}

type InsertActionInput struct {
	VAID               string
	ActorUserID        *string
	LiveOrganizationID *string
	ActionName         string
	Outcome            string
	UpstreamStatus     *int
	ErrorMessage       *string
	Meta               map[string]any
	StartedAt          time.Time
	FinishedAt         time.Time
}

func (r *ActionHistoryRepository) Insert(ctx context.Context, input InsertActionInput) error {
	meta := input.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal action meta: %w", err)
	}
	durationMS := int(input.FinishedAt.Sub(input.StartedAt).Milliseconds())
	if durationMS < 0 {
		durationMS = 0
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO public.live_api_v3_action_history (
    va_id, actor_user_id, live_organization_id, action_name, outcome,
    upstream_status, error_message, meta, started_at, finished_at, duration_ms
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		input.VAID, nullStringPtr(input.ActorUserID), nullStringPtr(input.LiveOrganizationID),
		input.ActionName, input.Outcome, nullIntPtr(input.UpstreamStatus), nullStringPtr(input.ErrorMessage),
		metaJSON, input.StartedAt.UTC(), input.FinishedAt.UTC(), durationMS,
	)
	if err != nil {
		return fmt.Errorf("insert live api v3 action history: %w", err)
	}
	return nil
}

func nullStringPtr(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func nullIntPtr(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
