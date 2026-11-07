package valive

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// CrewAssignmentRow is one assign_va_live_schedule_crew history entry for staff dashboards.
type CrewAssignmentRow struct {
	ID                 string
	StartedAt          time.Time
	Outcome            string
	UpstreamStatus     *int
	ErrorMessage       *string
	LiveOrganizationID *string
	ActorUsername      *string
	ActorIFCUsername   *string
	Meta               map[string]any
}

func (r *ActionHistoryRepository) ListCrewAssignments(ctx context.Context, vaID string) ([]CrewAssignmentRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("action history repository is not configured")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT h.id, h.started_at, h.outcome, h.upstream_status, h.error_message, h.meta,
       h.live_organization_id::text, u.username, u.if_community_id
FROM public.live_api_v3_action_history h
LEFT JOIN public.users u ON u.id = h.actor_user_id
WHERE h.va_id = $1 AND h.action_name = $2
ORDER BY h.started_at DESC`,
		vaID, ActionAssignScheduleCrew,
	)
	if err != nil {
		return nil, fmt.Errorf("list crew assignment history: %w", err)
	}
	defer rows.Close()

	out := make([]CrewAssignmentRow, 0)
	for rows.Next() {
		var row CrewAssignmentRow
		var upstream sql.NullInt64
		var errMsg, orgID, username, ifc sql.NullString
		var metaJSON []byte
		if err := rows.Scan(
			&row.ID, &row.StartedAt, &row.Outcome, &upstream, &errMsg, &metaJSON,
			&orgID, &username, &ifc,
		); err != nil {
			return nil, fmt.Errorf("scan crew assignment history: %w", err)
		}
		if upstream.Valid {
			v := int(upstream.Int64)
			row.UpstreamStatus = &v
		}
		row.ErrorMessage = nullStringVal(errMsg)
		row.LiveOrganizationID = nullStringVal(orgID)
		row.ActorUsername = nullStringVal(username)
		row.ActorIFCUsername = nullStringVal(ifc)
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &row.Meta)
		}
		if row.Meta == nil {
			row.Meta = map[string]any{}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func nullStringVal(v sql.NullString) *string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	s := strings.TrimSpace(v.String)
	return &s
}
