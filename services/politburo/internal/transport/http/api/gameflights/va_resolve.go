package gameflights

import (
	"context"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
)

// vaCodeResolver looks up a virtual airline by its code.
type vaCodeResolver interface {
	GetByCode(ctx context.Context, code string) (*virtualairlines.VirtualAirline, error)
}

func resolveVACode(ctx context.Context, resolver vaCodeResolver, code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return "", nil
	}
	va, err := resolver.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}
	if va == nil {
		return "", domainflights.ErrUnknownVACode
	}
	return va.ID, nil
}
