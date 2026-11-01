package livev3token

import (
	"context"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/livegame/valive"
)

// Job refreshes VA Live API grants whose access tokens expire soon.
type Job struct {
	grants   *valive.GrantsRepository
	provider *valive.TokenProvider
}

func New(grants *valive.GrantsRepository, provider *valive.TokenProvider) *Job {
	return &Job{grants: grants, provider: provider}
}

func (j *Job) Name() string {
	return "va_live_token_refresh"
}

func (j *Job) Run(ctx context.Context) error {
	if j.grants == nil || j.provider == nil {
		return nil
	}
	grants, err := j.grants.ListExpiringActive(ctx, time.Now())
	if err != nil {
		return err
	}
	for _, grant := range grants {
		if err := j.provider.RefreshGrant(ctx, grant.VAID); err != nil {
			slog.Warn("refresh va live oauth grant", "va_id", grant.VAID, "error", err)
		}
	}
	return nil
}
