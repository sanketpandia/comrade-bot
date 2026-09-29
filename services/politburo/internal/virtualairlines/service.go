package virtualairlines

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"infinite-experiment/politburo/internal/users"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrServerAlreadyVA    = errors.New("server already registered")
	ErrCodeTaken          = errors.New("va code taken")
	ErrInvalidVACode      = errors.New("invalid va code")
)

type Service struct {
	db    *sql.DB
	users *users.Repository
	vas   *Repository
}

func NewService(db *sql.DB, users *users.Repository, vas *Repository) *Service {
	return &Service{db: db, users: users, vas: vas}
}

type InitInput struct {
	DiscordUserID   string
	DiscordServerID string
	VACode          string
	DisplayName     string
}

type InitResult struct {
	VA           VirtualAirline
	SetupRequired bool
}

func (s *Service) InitServer(ctx context.Context, input InitInput) (*InitResult, error) {
	code := strings.TrimSpace(input.VACode)
	if len(code) < 2 || len(code) > 30 {
		return nil, ErrInvalidVACode
	}

	user, err := s.users.GetByDiscordID(ctx, input.DiscordUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	existingServer, err := s.vas.GetByDiscordServerID(ctx, input.DiscordServerID)
	if err != nil {
		return nil, err
	}
	if existingServer != nil {
		return nil, ErrServerAlreadyVA
	}

	existingCode, err := s.vas.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existingCode != nil {
		return nil, ErrCodeTaken
	}

	name := strings.TrimSpace(input.DisplayName)
	if name == "" {
		name = code
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	vaRow := tx.QueryRowContext(ctx, `
INSERT INTO public.virtual_airlines (name, code, discord_server_id, is_active)
VALUES ($1, $2, $3, true)
RETURNING id, name, code, discord_server_id, is_active, created_at, updated_at`,
		name, code, input.DiscordServerID,
	)
	va, err := scanVA(vaRow)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.va_user_roles (user_id, va_id, role, is_active)
VALUES ($1, $2, 'administrator'::public.va_role, true)`,
		user.ID, va.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("create administrator membership: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &InitResult{VA: *va, SetupRequired: true}, nil
}

func (s *Service) RebindDiscordServer(ctx context.Context, vaID, newDiscordServerID string) error {
	return s.vas.RebindDiscordServer(ctx, vaID, newDiscordServerID)
}
