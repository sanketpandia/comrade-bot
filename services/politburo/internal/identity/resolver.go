package identity

import (
	"context"

	"infinite-experiment/politburo/internal/membership"
	"infinite-experiment/politburo/internal/users"
	"infinite-experiment/politburo/internal/virtualairlines"
)

type Resolver struct {
	users       *users.Repository
	vas         *virtualairlines.Repository
	memberships *membership.Repository
}

func NewResolver(users *users.Repository, vas *virtualairlines.Repository, memberships *membership.Repository) *Resolver {
	return &Resolver{users: users, vas: vas, memberships: memberships}
}

func (r *Resolver) ResolveMembership(ctx context.Context, discordUserID, discordServerID string) (vaID, role string, ok bool, err error) {
	va, err := r.vas.GetByDiscordServerID(ctx, discordServerID)
	if err != nil || va == nil {
		return "", "", false, err
	}
	user, err := r.users.GetByDiscordID(ctx, discordUserID)
	if err != nil || user == nil {
		return "", "", false, err
	}
	m, err := r.memberships.GetActiveForUserAndVA(ctx, user.ID, va.ID)
	if err != nil || m == nil {
		return "", "", false, err
	}
	return va.ID, m.Role, true, nil
}
