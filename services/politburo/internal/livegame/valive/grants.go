package valive

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	GrantStatusActive          = "active"
	GrantStatusNeedsReconnect  = "needs_reconnect"
	grantRefreshLeadTime       = 3 * time.Minute
)

// Grant tracks VA-scoped Live API OAuth metadata (secrets live in Redis).
type Grant struct {
	VAID               string
	ConnectedByUserID  string
	Status             string
	AccessExpiresAt    time.Time
	LastRefreshError   *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type GrantsRepository struct {
	db *sql.DB
}

func NewGrantsRepository(db *sql.DB) *GrantsRepository {
	return &GrantsRepository{db: db}
}

const grantColumns = `va_id, connected_by_user_id, status, access_expires_at, last_refresh_error, created_at, updated_at`

func (r *GrantsRepository) UpsertActive(ctx context.Context, vaID, userID string, accessExpiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO public.va_live_grants (va_id, connected_by_user_id, status, access_expires_at, last_refresh_error, updated_at)
VALUES ($1, $2, $3, $4, NULL, now() AT TIME ZONE 'UTC')
ON CONFLICT (va_id) DO UPDATE SET
    connected_by_user_id = EXCLUDED.connected_by_user_id,
    status = $3,
    access_expires_at = EXCLUDED.access_expires_at,
    last_refresh_error = NULL,
    updated_at = now() AT TIME ZONE 'UTC'`,
		vaID, userID, GrantStatusActive, accessExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("upsert va live grant: %w", err)
	}
	return nil
}

func (r *GrantsRepository) UpdateAccessExpiry(ctx context.Context, vaID string, accessExpiresAt time.Time) error {
	res, err := r.db.ExecContext(ctx, `
UPDATE public.va_live_grants
SET access_expires_at = $2, status = $3, last_refresh_error = NULL, updated_at = now() AT TIME ZONE 'UTC'
WHERE va_id = $1`,
		vaID, accessExpiresAt, GrantStatusActive,
	)
	if err != nil {
		return fmt.Errorf("update va live grant expiry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *GrantsRepository) MarkNeedsReconnect(ctx context.Context, vaID string, refreshError string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE public.va_live_grants
SET status = $2, last_refresh_error = $3, updated_at = now() AT TIME ZONE 'UTC'
WHERE va_id = $1`,
		vaID, GrantStatusNeedsReconnect, refreshError,
	)
	if err != nil {
		return fmt.Errorf("mark va live grant needs reconnect: %w", err)
	}
	return nil
}

func (r *GrantsRepository) GetByVAID(ctx context.Context, vaID string) (*Grant, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+grantColumns+` FROM public.va_live_grants WHERE va_id = $1`, vaID)
	grant, err := scanGrant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return grant, err
}

// ListExpiringActive returns active grants whose access token expires within the refresh lead window.
func (r *GrantsRepository) ListExpiringActive(ctx context.Context, now time.Time) ([]Grant, error) {
	deadline := now.UTC().Add(grantRefreshLeadTime)
	rows, err := r.db.QueryContext(ctx, `
SELECT `+grantColumns+`
FROM public.va_live_grants
WHERE status = $1 AND access_expires_at <= $2
ORDER BY access_expires_at`,
		GrantStatusActive, deadline,
	)
	if err != nil {
		return nil, fmt.Errorf("list expiring va live grants: %w", err)
	}
	defer rows.Close()
	var out []Grant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

func scanGrant(row interface{ Scan(dest ...any) error }) (*Grant, error) {
	var g Grant
	var lastErr sql.NullString
	if err := row.Scan(
		&g.VAID, &g.ConnectedByUserID, &g.Status, &g.AccessExpiresAt, &lastErr, &g.CreatedAt, &g.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan va live grant: %w", err)
	}
	if lastErr.Valid {
		s := lastErr.String
		g.LastRefreshError = &s
	}
	return &g, nil
}
