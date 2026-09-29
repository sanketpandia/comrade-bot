package identity

import (
	"context"

	"infinite-experiment/politburo/internal/virtualairlines"
)

type VALookup struct {
	vas *virtualairlines.Repository
}

func NewVALookup(vas *virtualairlines.Repository) *VALookup {
	return &VALookup{vas: vas}
}

func (l *VALookup) IsConfiguredVA(ctx context.Context, discordServerID string) (bool, error) {
	va, err := l.vas.GetByDiscordServerID(ctx, discordServerID)
	if err != nil {
		return false, err
	}
	return va != nil, nil
}
