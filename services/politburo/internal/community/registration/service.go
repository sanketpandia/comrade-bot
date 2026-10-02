package registration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"infinite-experiment/politburo/internal/community/registration/proof"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/community/users"
)

var (
	ErrBanned              = errors.New("discord id is banned")
	ErrAlreadyRegistered   = errors.New("user already registered")
	ErrIFCAlreadyLinked    = errors.New("ifc already linked")
	ErrFlightProofFailed   = errors.New("flight proof failed")
	ErrIFUserNotFound      = errors.New("if user not found")
	ErrIFClientUnavailable = errors.New("infinite flight client unavailable")
)

type IFUsers interface {
	ResolveUserIDByIFCUsername(ctx context.Context, ifcUsername string) (string, error)
	RecentLogbookFlights(ctx context.Context, userID string, limit int) ([]infiniteflight.LogbookFlight, error)
}

type Service struct {
	users *users.Repository
	ifc   IFUsers
}

func NewService(users *users.Repository, ifc IFUsers) *Service {
	return &Service{users: users, ifc: ifc}
}

type RegisterInput struct {
	DiscordID   string
	IFCUsername string
	RouteProof  string
}

type RegisterResult struct {
	User *users.User
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (result *RegisterResult, err error) {
	discordID := strings.TrimSpace(input.DiscordID)
	ifc := strings.TrimSpace(input.IFCUsername)
	defer func() { logUserRegistration(ctx, discordID, ifc, err) }()

	if discordID == "" || ifc == "" {
		return nil, fmt.Errorf("discord id and ifc username are required")
	}
	if s.ifc == nil {
		return nil, ErrIFClientUnavailable
	}

	banned, err := s.users.IsDiscordBanned(ctx, discordID)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrBanned
	}

	existing, err := s.users.GetByDiscordID(ctx, discordID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyRegistered
	}

	occupant, err := s.users.GetByIFCommunityID(ctx, ifc)
	if err != nil {
		return nil, err
	}
	if occupant != nil && occupant.DiscordID != discordID {
		return nil, ErrIFCAlreadyLinked
	}

	origin, dest, err := proof.ParseRouteProof(input.RouteProof)
	if err != nil {
		return nil, err
	}

	ifUserID, err := s.ifc.ResolveUserIDByIFCUsername(ctx, ifc)
	if err != nil {
		if errors.Is(err, infiniteflight.ErrUserNotFound) {
			return nil, ErrIFUserNotFound
		}
		return nil, err
	}

	flights, err := s.ifc.RecentLogbookFlights(ctx, ifUserID, 0)
	if err != nil {
		return nil, err
	}
	if !proof.MatchesLatestCompleteFlight(flights, origin, dest) {
		return nil, ErrFlightProofFailed
	}

	user, err := s.users.Create(ctx, users.CreateInput{
		DiscordID:     discordID,
		IFCommunityID: ifc,
		IFAPIID:       ifUserID,
		Username:      ifc,
	})
	if err != nil {
		return nil, err
	}
	return &RegisterResult{User: user}, nil
}

func logUserRegistration(ctx context.Context, discordID, ifc string, err error) {
	args := []any{
		"discord_id", discordID,
		"ifc_username", ifc,
		"result", registerResult(err),
	}
	if requestID := chimiddleware.GetReqID(ctx); requestID != "" {
		args = append(args, "request_id", requestID)
	}
	if err == nil || registerResult(err) != "failed" {
		slog.Info("user_registration", args...)
		return
	}
	args = append(args, "error", err)
	slog.Error("user_registration", args...)
}

func registerResult(err error) string {
	if err == nil {
		return "created"
	}
	switch {
	case errors.Is(err, ErrBanned):
		return "banned"
	case errors.Is(err, ErrAlreadyRegistered):
		return "already_registered"
	case errors.Is(err, ErrIFCAlreadyLinked):
		return "ifc_linked"
	case errors.Is(err, ErrFlightProofFailed):
		return "proof_failed"
	case errors.Is(err, ErrIFUserNotFound):
		return "if_user_not_found"
	case errors.Is(err, ErrIFClientUnavailable):
		return "if_unavailable"
	case errors.Is(err, proof.ErrInvalidRoute):
		return "invalid_route"
	default:
		return "failed"
	}
}
