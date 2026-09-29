package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const userColumns = `id, discord_id, if_community_id, if_api_id, is_active, username, created_at, updated_at`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByDiscordID(ctx context.Context, discordID string) (*User, error) {
	return r.queryOne(ctx, `SELECT `+userColumns+` FROM public.users WHERE discord_id = $1`, discordID)
}

func (r *Repository) GetByIFCommunityID(ctx context.Context, ifc string) (*User, error) {
	return r.queryOne(ctx, `SELECT `+userColumns+` FROM public.users WHERE if_community_id = $1`, ifc)
}

func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	return r.queryOne(ctx, `SELECT `+userColumns+` FROM public.users WHERE id = $1`, id)
}

func (r *Repository) IsDiscordBanned(ctx context.Context, discordID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM public.banned_discord_ids WHERE discord_id = $1)`,
		discordID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check banned discord id: %w", err)
	}
	return exists, nil
}

type CreateInput struct {
	DiscordID     string
	IFCommunityID string
	IFAPIID       string
	Username      string
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (*User, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.users (discord_id, if_community_id, if_api_id, is_active, username)
VALUES ($1, $2, $3, true, $4)
RETURNING `+userColumns,
		input.DiscordID, input.IFCommunityID, nullUUID(input.IFAPIID), input.Username,
	)
	return scanUser(row)
}

func (r *Repository) Delete(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM public.va_user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete memberships: %w", err)
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM public.users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) queryOne(ctx context.Context, query string, arg string) (*User, error) {
	row := r.db.QueryRowContext(ctx, query, arg)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func scanUser(row interface {
	Scan(dest ...any) error
}) (*User, error) {
	var user User
	var ifCommunityID, ifAPIID, username sql.NullString
	err := row.Scan(
		&user.ID,
		&user.DiscordID,
		&ifCommunityID,
		&ifAPIID,
		&user.IsActive,
		&username,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	user.IFCommunityID = nullString(ifCommunityID)
	user.IFApiID = nullString(ifAPIID)
	user.Username = nullString(username)
	return &user, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}

func nullUUID(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
