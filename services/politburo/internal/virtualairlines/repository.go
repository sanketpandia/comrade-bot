package virtualairlines

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const vaColumns = `id, name, code, discord_server_id, is_active, created_at, updated_at`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByDiscordServerID(ctx context.Context, discordServerID string) (*VirtualAirline, error) {
	return r.queryOne(ctx, `SELECT `+vaColumns+` FROM public.virtual_airlines WHERE discord_server_id = $1`, discordServerID)
}

func (r *Repository) GetByID(ctx context.Context, id string) (*VirtualAirline, error) {
	return r.queryOne(ctx, `SELECT `+vaColumns+` FROM public.virtual_airlines WHERE id = $1`, id)
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*VirtualAirline, error) {
	return r.queryOne(ctx, `SELECT `+vaColumns+` FROM public.virtual_airlines WHERE lower(code) = lower($1)`, code)
}

type CreateInput struct {
	Name            string
	Code            string
	DiscordServerID string
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (*VirtualAirline, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.virtual_airlines (name, code, discord_server_id, is_active)
VALUES ($1, $2, $3, true)
RETURNING `+vaColumns,
		input.Name, input.Code, input.DiscordServerID,
	)
	return scanVA(row)
}

func (r *Repository) RebindDiscordServer(ctx context.Context, vaID, newDiscordServerID string) error {
	res, err := r.db.ExecContext(ctx, `
UPDATE public.virtual_airlines
SET discord_server_id = $2, updated_at = now()
WHERE id = $1`,
		vaID, newDiscordServerID,
	)
	if err != nil {
		return fmt.Errorf("rebind discord server: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) queryOne(ctx context.Context, query string, arg string) (*VirtualAirline, error) {
	row := r.db.QueryRowContext(ctx, query, arg)
	va, err := scanVA(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return va, err
}

func scanVA(row interface{ Scan(dest ...any) error }) (*VirtualAirline, error) {
	var va VirtualAirline
	var discordServerID sql.NullString
	err := row.Scan(
		&va.ID, &va.Name, &va.Code, &discordServerID, &va.IsActive, &va.CreatedAt, &va.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan virtual airline: %w", err)
	}
	va.DiscordServerID = nullString(discordServerID)
	return &va, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}
