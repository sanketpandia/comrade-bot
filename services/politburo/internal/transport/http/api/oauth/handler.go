package oauth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/access/session"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/routes"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type tokenCache interface {
	SetJSON(context.Context, string, any, time.Duration) error
	GetDelJSON(context.Context, string, any) error
}

// Handler serves Politburo Infinite Flight OAuth start and callback routes.
type Handler struct {
	client      *infiniteflight.OAuthClient
	redirectURI string
	cache       tokenCache
}

func NewHandler(client *infiniteflight.OAuthClient, redirectURI string, cache tokenCache) *Handler {
	return &Handler{client: client, redirectURI: redirectURI, cache: cache}
}

// GetAuthorizeURL returns the Infinite Flight authorize URL for the signed-in user.
// Caller auth is handled by AuthenticateAPI (session cookie or API key on /api/v1).
func (h *Handler) GetAuthorizeURL(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "OAUTH_DISABLED", "Infinite Flight OAuth is not configured")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || strings.TrimSpace(claims.PbUserID) == "" {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sign in required")
		return
	}
	userID := strings.TrimSpace(claims.PbUserID)
	state, err := infiniteflight.NewOAuthState()
	if err != nil {
		slog.Error("generate oauth state", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start OAuth")
		return
	}
	verifier, challenge, err := infiniteflight.NewPKCEPair()
	if err != nil {
		slog.Error("generate oauth pkce", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start OAuth")
		return
	}
	pending := infiniteflight.OAuthPending{
		UserID:       userID,
		CodeVerifier: verifier,
	}
	if err := h.cache.SetJSON(r.Context(), cache.KeyIFOAuthPending(state), pending, infiniteflight.OAuthPendingTTL); err != nil {
		slog.Error("store oauth pending session", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start OAuth")
		return
	}
	authorizeURL, err := h.client.AuthorizationURL(h.redirectURI, state, challenge, infiniteflight.DefaultOAuthScopes)
	if err != nil {
		slog.Error("build oauth authorize url", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start OAuth")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"url": authorizeURL},
	})
}

// CompleteCallback exchanges the authorization code and stores tokens in Redis.
func (h *Handler) CompleteCallback(w http.ResponseWriter, r *http.Request, code, state string) {
	if h.client == nil {
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	state = strings.TrimSpace(state)
	code = strings.TrimSpace(code)
	if state == "" || code == "" {
		slog.Info("oauth callback missing parameters")
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	var pending infiniteflight.OAuthPending
	if err := h.cache.GetDelJSON(r.Context(), cache.KeyIFOAuthPending(state), &pending); err != nil {
		if !errors.Is(err, cache.ErrMiss) {
			slog.Error("load oauth pending session", "error", err)
		} else {
			slog.Info("oauth callback unknown or expired state")
		}
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	userID := strings.TrimSpace(pending.UserID)
	if userID == "" {
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	if sessionUserID := strings.TrimSpace(claims.PbUserID); sessionUserID != "" && sessionUserID != userID {
		slog.Info("oauth callback user mismatch")
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	tokens, err := h.client.ExchangeCode(r.Context(), h.redirectURI, code, pending.CodeVerifier)
	if err != nil {
		slog.Error("exchange oauth code", "error", err)
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	expiresAt := time.Now().UTC().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	if tokens.ExpiresIn <= 0 {
		expiresAt = time.Now().UTC().Add(time.Hour)
	}
	stored := infiniteflight.OAuthTokenSession{
		AccessToken:     tokens.AccessToken,
		RefreshToken:    tokens.RefreshToken,
		TokenType:       tokens.TokenType,
		Scope:           tokens.Scope,
		AccessExpiresAt: expiresAt,
	}
	if err := h.cache.SetJSON(r.Context(), cache.KeyIFOAuthUser(userID), stored, session.TTL); err != nil {
		slog.Error("store oauth token session", "error", err)
		http.Redirect(w, r, routes.Dashboard+"?oauth=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, routes.Dashboard+"?oauth=connected", http.StatusSeeOther)
}
