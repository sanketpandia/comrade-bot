package registration

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (s *Service) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	discordID := strings.TrimSpace(input.DiscordID)
	ifc := strings.TrimSpace(input.IFCUsername)
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
