package operator

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"infinite-experiment/politburo/internal/membership"
	"infinite-experiment/politburo/internal/reports"
	"infinite-experiment/politburo/internal/users"
	"infinite-experiment/politburo/internal/virtualairlines"
)

var ErrOccupantNotFound = errors.New("occupant not found")

type Service struct {
	db          *sql.DB
	users       *users.Repository
	memberships *membership.Repository
	reports     *reports.Repository
	vas         *virtualairlines.Repository
}

func NewService(db *sql.DB, users *users.Repository, memberships *membership.Repository, reportRepo *reports.Repository, vas *virtualairlines.Repository) *Service {
	return &Service{db: db, users: users, memberships: memberships, reports: reportRepo, vas: vas}
}

func (s *Service) ResolveOccupiedIFC(ctx context.Context, reportID, operatorDiscordID string) error {
	report, err := s.reports.GetByID(ctx, reportID)
	if err != nil {
		return err
	}
	if report == nil || report.Kind != reports.KindOccupiedIFC {
		return sql.ErrNoRows
	}
	if report.ClaimedIFC == nil {
		return fmt.Errorf("report missing claimed ifc")
	}

	occupant, err := s.users.GetByIFCommunityID(ctx, *report.ClaimedIFC)
	if err != nil {
		return err
	}
	if occupant == nil {
		return ErrOccupantNotFound
	}

	if err := s.deleteUserWithArchive(ctx, occupant, operatorDiscordID); err != nil {
		return err
	}
	return s.reports.MarkResolved(ctx, reportID, operatorDiscordID)
}

func (s *Service) ResolveGuildMigration(ctx context.Context, reportID, operatorDiscordID string) error {
	report, err := s.reports.GetByID(ctx, reportID)
	if err != nil {
		return err
	}
	if report == nil || report.Kind != reports.KindMigrateDiscordServer {
		return sql.ErrNoRows
	}
	if report.VAID == nil || report.NewDiscordServerID == nil {
		return fmt.Errorf("report missing va or guild id")
	}
	if err := s.vas.RebindDiscordServer(ctx, *report.VAID, *report.NewDiscordServerID); err != nil {
		return err
	}
	return s.reports.MarkResolved(ctx, reportID, operatorDiscordID)
}

func (s *Service) deleteUserWithArchive(ctx context.Context, user *users.User, operatorDiscordID string) error {
	memberships, err := s.memberships.ListForUser(ctx, user.ID)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"user":        user,
		"memberships": memberships,
		"archived_at": time.Now().UTC(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.user_deletion_archives (deleted_by, payload) VALUES ($1, $2)`,
		operatorDiscordID, raw,
	)
	if err != nil {
		return fmt.Errorf("insert archive: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.banned_discord_ids (discord_id, banned_by, reason)
VALUES ($1, $2, $3)
ON CONFLICT (discord_id) DO NOTHING`,
		user.DiscordID, operatorDiscordID, "occupied ifc takedown",
	)
	if err != nil {
		return fmt.Errorf("ban discord id: %w", err)
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM public.va_user_roles WHERE user_id = $1`, user.ID)
	if err != nil {
		return fmt.Errorf("delete memberships: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM public.users WHERE id = $1`, user.ID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}
