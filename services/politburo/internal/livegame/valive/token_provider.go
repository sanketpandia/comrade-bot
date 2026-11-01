package valive

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

var ErrOAuthRequired = errors.New("infinite flight oauth required for VA")

type tokenStore interface {
	GetJSON(context.Context, string, any) error
	SetJSON(context.Context, string, any, time.Duration) error
}

// TokenProvider resolves VA-scoped Live API access tokens, refreshing when expired.
type TokenProvider struct {
	oauth   *infiniteflight.OAuthClient
	cache   tokenStore
	grants  *GrantsRepository
	actions *ActionHistoryRepository
}

func NewTokenProvider(oauth *infiniteflight.OAuthClient, cache tokenStore, grants *GrantsRepository, actions *ActionHistoryRepository) *TokenProvider {
	return &TokenProvider{oauth: oauth, cache: cache, grants: grants, actions: actions}
}

func AccessExpiresAt(tokens infiniteflight.OAuthTokens) time.Time {
	expiresIn := tokens.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
}

func accessTokenExpired(expiresAt time.Time) bool {
	return !expiresAt.IsZero() && !time.Now().UTC().Before(expiresAt)
}

// StoreVATokens writes token secrets to Redis and grant metadata to Postgres.
func (p *TokenProvider) StoreVATokens(ctx context.Context, vaID, userID string, tokens infiniteflight.OAuthTokens) error {
	expiresAt := AccessExpiresAt(tokens)
	stored := infiniteflight.OAuthTokenSession{
		AccessToken:     tokens.AccessToken,
		RefreshToken:    tokens.RefreshToken,
		TokenType:       tokens.TokenType,
		Scope:           tokens.Scope,
		AccessExpiresAt: expiresAt,
	}
	// TODO: encrypt OAuth tokens at rest in Redis before a dedicated encryption feature ships.
	if err := p.cache.SetJSON(ctx, cache.KeyIFOAuthVA(vaID), stored, infiniteflight.VaLiveOAuthTokenTTL); err != nil {
		return fmt.Errorf("store va oauth tokens: %w", err)
	}
	if err := p.grants.UpsertActive(ctx, vaID, userID, expiresAt); err != nil {
		return err
	}
	return nil
}

// AccessToken returns a valid access token for the VA, refreshing once if needed.
func (p *TokenProvider) AccessToken(ctx context.Context, vaID string) (string, error) {
	if p.oauth == nil {
		return "", ErrOAuthRequired
	}
	var stored infiniteflight.OAuthTokenSession
	err := p.cache.GetJSON(ctx, cache.KeyIFOAuthVA(vaID), &stored)
	if errors.Is(err, cache.ErrMiss) {
		return "", ErrOAuthRequired
	}
	if err != nil {
		return "", fmt.Errorf("load va oauth tokens: %w", err)
	}
	if !accessTokenExpired(stored.AccessExpiresAt) {
		return stored.AccessToken, nil
	}
	if err := p.refreshAndStore(ctx, vaID, stored.RefreshToken); err != nil {
		return "", err
	}
	if err := p.cache.GetJSON(ctx, cache.KeyIFOAuthVA(vaID), &stored); err != nil {
		return "", fmt.Errorf("reload va oauth tokens: %w", err)
	}
	return stored.AccessToken, nil
}

// RefreshGrant refreshes tokens for a VA grant (used by the scheduled job).
func (p *TokenProvider) RefreshGrant(ctx context.Context, vaID string) error {
	if p.oauth == nil {
		return ErrOAuthRequired
	}
	var stored infiniteflight.OAuthTokenSession
	err := p.cache.GetJSON(ctx, cache.KeyIFOAuthVA(vaID), &stored)
	if errors.Is(err, cache.ErrMiss) {
		_ = p.grants.MarkNeedsReconnect(ctx, vaID, "redis token missing")
		return ErrOAuthRequired
	}
	if err != nil {
		return fmt.Errorf("load va oauth tokens: %w", err)
	}
	return p.refreshAndStore(ctx, vaID, stored.RefreshToken)
}

func (p *TokenProvider) refreshAndStore(ctx context.Context, vaID, refreshToken string) error {
	tokens, err := p.oauth.Refresh(ctx, refreshToken)
	if err != nil {
		refreshErr := err.Error()
		if markErr := p.grants.MarkNeedsReconnect(ctx, vaID, refreshErr); markErr != nil {
			slog.Error("mark va live grant needs reconnect", "va_id", vaID, "error", markErr)
		}
		return fmt.Errorf("refresh va oauth token: %w", err)
	}
	expiresAt := AccessExpiresAt(tokens)
	stored := infiniteflight.OAuthTokenSession{
		AccessToken:     tokens.AccessToken,
		RefreshToken:    tokens.RefreshToken,
		TokenType:       tokens.TokenType,
		Scope:           tokens.Scope,
		AccessExpiresAt: expiresAt,
	}
	// TODO: encrypt OAuth tokens at rest in Redis before a dedicated encryption feature ships.
	if err := p.cache.SetJSON(ctx, cache.KeyIFOAuthVA(vaID), stored, infiniteflight.VaLiveOAuthTokenTTL); err != nil {
		return fmt.Errorf("store refreshed va oauth tokens: %w", err)
	}
	return p.grants.UpdateAccessExpiry(ctx, vaID, expiresAt)
}

// Write runs a Live API v3 write call with a resolved VA token and records action history.
func (p *TokenProvider) Write(ctx context.Context, vaID string, actorUserID *string, orgID *string, actionName string, fn func(accessToken string) error) error {
	started := time.Now().UTC()
	accessToken, err := p.AccessToken(ctx, vaID)
	if err != nil {
		return err
	}
	callErr := fn(accessToken)
	finished := time.Now().UTC()
	outcome := ActionOutcomeSuccess
	var errMsg *string
	if callErr != nil {
		outcome = ActionOutcomeError
		msg := callErr.Error()
		errMsg = &msg
	}
	if p.actions != nil {
		_ = p.actions.Insert(ctx, InsertActionInput{
			VAID:               vaID,
			ActorUserID:        actorUserID,
			LiveOrganizationID: orgID,
			ActionName:         actionName,
			Outcome:            outcome,
			ErrorMessage:       errMsg,
			StartedAt:          started,
			FinishedAt:         finished,
		})
	}
	return callErr
}
