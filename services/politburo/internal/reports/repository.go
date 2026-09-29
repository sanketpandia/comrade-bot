package reports

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOccupiedIFC(ctx context.Context, reporterDiscordID, claimedIFC, note string) (*Report, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.platform_reports (kind, status, reporter_discord_id, claimed_ifc, note)
VALUES ($1, $2, $3, $4, nullif($5, ''))
RETURNING id, kind, status, reporter_discord_id, claimed_ifc, note, va_id, new_discord_server_id, created_at, resolved_at, resolved_by`,
		KindOccupiedIFC, StatusOpen, reporterDiscordID, claimedIFC, note,
	)
	return scanReport(row)
}

func (r *Repository) CreateGuildMigration(ctx context.Context, reporterDiscordID, vaID, newDiscordServerID, note string) (*Report, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.platform_reports (kind, status, reporter_discord_id, va_id, new_discord_server_id, note)
VALUES ($1, $2, $3, $4, $5, nullif($6, ''))
RETURNING id, kind, status, reporter_discord_id, claimed_ifc, note, va_id, new_discord_server_id, created_at, resolved_at, resolved_by`,
		KindMigrateDiscordServer, StatusOpen, reporterDiscordID, vaID, newDiscordServerID, note,
	)
	return scanReport(row)
}

func (r *Repository) ListOpen(ctx context.Context) ([]Report, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, kind, status, reporter_discord_id, claimed_ifc, note, va_id, new_discord_server_id, created_at, resolved_at, resolved_by
FROM public.platform_reports
WHERE status = $1
ORDER BY created_at ASC`,
		StatusOpen,
	)
	if err != nil {
		return nil, fmt.Errorf("list open reports: %w", err)
	}
	defer rows.Close()
	var out []Report
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rep)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Report, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, kind, status, reporter_discord_id, claimed_ifc, note, va_id, new_discord_server_id, created_at, resolved_at, resolved_by
FROM public.platform_reports WHERE id = $1`, id)
	rep, err := scanReport(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return rep, err
}

func (r *Repository) MarkResolved(ctx context.Context, id, resolvedBy string) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
UPDATE public.platform_reports
SET status = $2, resolved_at = $3, resolved_by = $4
WHERE id = $1 AND status = $5`,
		id, StatusResolved, now, resolvedBy, StatusOpen,
	)
	if err != nil {
		return fmt.Errorf("resolve report: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanReport(row interface{ Scan(dest ...any) error }) (*Report, error) {
	var rep Report
	var claimedIFC, note, vaID, newServer, resolvedBy sql.NullString
	var resolvedAt sql.NullTime
	err := row.Scan(
		&rep.ID, &rep.Kind, &rep.Status, &rep.ReporterDiscordID,
		&claimedIFC, &note, &vaID, &newServer, &rep.CreatedAt, &resolvedAt, &resolvedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("scan report: %w", err)
	}
	rep.ClaimedIFC = nullString(claimedIFC)
	rep.Note = nullString(note)
	rep.VAID = nullString(vaID)
	rep.NewDiscordServerID = nullString(newServer)
	rep.ResolvedBy = nullString(resolvedBy)
	if resolvedAt.Valid {
		t := resolvedAt.Time
		rep.ResolvedAt = &t
	}
	return &rep, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}
