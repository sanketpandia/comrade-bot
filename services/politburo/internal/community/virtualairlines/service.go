package virtualairlines

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"infinite-experiment/politburo/internal/community/users"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrServerAlreadyVA = errors.New("server already registered")
	ErrCodeTaken       = errors.New("va code taken")
	ErrInvalidVACode   = errors.New("invalid va code")
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
	VA            VirtualAirline
	SetupRequired bool
}

func (s *Service) InitServer(ctx context.Context, input InitInput) (result *InitResult, err error) {
	discordUserID := strings.TrimSpace(input.DiscordUserID)
	discordServerID := strings.TrimSpace(input.DiscordServerID)
	code := strings.ToUpper(strings.TrimSpace(input.VACode))
	defer func() {
		logVAServerInit(ctx, discordUserID, discordServerID, code, result, err)
	}()

	if !isValidVACode(code) {
		return nil, ErrInvalidVACode
	}

	user, err := s.users.GetByDiscordID(ctx, discordUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	existingServer, err := s.vas.GetByDiscordServerID(ctx, discordServerID)
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
		name, code, discordServerID,
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

func isValidVACode(code string) bool {
	if len(code) < 3 || len(code) > 5 {
		return false
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func logVAServerInit(ctx context.Context, discordUserID, discordServerID, vaCode string, result *InitResult, err error) {
	args := []any{
		"discord_user_id", discordUserID,
		"discord_server_id", discordServerID,
		"va_code", vaCode,
		"result", vaServerInitResult(result, err),
	}
	if requestID := chimiddleware.GetReqID(ctx); requestID != "" {
		args = append(args, "request_id", requestID)
	}
	if result != nil {
		args = append(args, "va_id", result.VA.ID)
	}
	if err == nil || vaServerInitResult(result, err) != "failed" {
		slog.Info("va_server_init", args...)
		return
	}
	args = append(args, "error", err)
	slog.Error("va_server_init", args...)
}

func vaServerInitResult(result *InitResult, err error) string {
	if err == nil && result != nil {
		return "created"
	}
	switch {
	case errors.Is(err, ErrUserNotFound):
		return "user_not_found"
	case errors.Is(err, ErrServerAlreadyVA):
		return "server_already_va"
	case errors.Is(err, ErrCodeTaken):
		return "code_taken"
	case errors.Is(err, ErrInvalidVACode):
		return "invalid_code"
	default:
		return "failed"
	}
}
