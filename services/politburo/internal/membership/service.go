package membership

import (
	"context"
	"errors"
	"strings"

	"infinite-experiment/politburo/internal/users"
	"infinite-experiment/politburo/internal/va/roster"
	"infinite-experiment/politburo/internal/virtualairlines"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrVANotFound        = errors.New("va not found")
	ErrAlreadyMember     = errors.New("already a member")
	ErrNotOnRoster       = errors.New("not on roster")
	ErrCallsignRequired  = errors.New("callsign required")
)

type Service struct {
	users       *users.Repository
	memberships *Repository
	vas         *virtualairlines.Repository
	roster      roster.Checker
}

func NewService(users *users.Repository, memberships *Repository, vas *virtualairlines.Repository, rosterChecker roster.Checker) *Service {
	if rosterChecker == nil {
		rosterChecker = roster.Noop{}
	}
	return &Service{users: users, memberships: memberships, vas: vas, roster: rosterChecker}
}

type JoinCurrentServerInput struct {
	DiscordID       string
	DiscordServerID string
	Callsign        string
}

type JoinResult struct {
	Membership Membership
}

func (s *Service) JoinCurrentServer(ctx context.Context, input JoinCurrentServerInput) (*JoinResult, error) {
	callsign := strings.TrimSpace(input.Callsign)
	if callsign == "" {
		return nil, ErrCallsignRequired
	}

	user, err := s.users.GetByDiscordID(ctx, input.DiscordID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	va, err := s.vas.GetByDiscordServerID(ctx, input.DiscordServerID)
	if err != nil {
		return nil, err
	}
	if va == nil {
		return nil, ErrVANotFound
	}

	existing, err := s.memberships.GetActiveForUserAndVA(ctx, user.ID, va.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	airtablePilotID := ""
	if user.IFCommunityID != nil {
		rosterResult, err := s.roster.LookupPilotByIFC(ctx, va.ID, *user.IFCommunityID)
		if err != nil {
			return nil, err
		}
		if rosterResult != nil {
			airtablePilotID = rosterResult.AirtablePilotID
		}
	}

	m, err := s.memberships.Join(ctx, JoinInput{
		UserID:          user.ID,
		VAID:            va.ID,
		Role:            RoleProletariat,
		Callsign:        callsign,
		AirtablePilotID: airtablePilotID,
	})
	if err != nil {
		return nil, err
	}
	return &JoinResult{Membership: *m}, nil
}
