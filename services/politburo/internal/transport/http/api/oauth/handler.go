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
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/routes"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type tokenCache interface {
	SetJSON(context.Context, string, any, time.Duration) error
	GetDelJSON(context.Context, string, any) error
}

type organizationsClient interface {
	ListOrganizations(context.Context, string) ([]infiniteflight.LiveOrganization, error)
}

type currentUser interface {
	Current(context.Context, auth.Claims) (*users.User, error)
}

// Handler serves Politburo Infinite Flight OAuth start and callback routes.
type Handler struct {
	client      *infiniteflight.OAuthClient
	redirectURI string
	cache       tokenCache
	users       currentUser
	live        organizationsClient
	tokens      *valive.TokenProvider
	connections *valive.ConnectionsRepository
}

func NewHandler(
	client *infiniteflight.OAuthClient,
	redirectURI string,
	cache tokenCache,
	users currentUser,
	live organizationsClient,
	tokens *valive.TokenProvider,
	connections *valive.ConnectionsRepository,
) *Handler {
	return &Handler{
		client: client, redirectURI: redirectURI, cache: cache, users: users,
		live: live, tokens: tokens, connections: connections,
	}
}

// GetAuthorizeURL returns the Infinite Flight authorize URL for the current user.
func (h *Handler) GetAuthorizeURL(w http.ResponseWriter, r *http.Request, scopeSet string) {
	if h.client == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "OAUTH_DISABLED", "Infinite Flight OAuth is not configured")
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	current, err := h.users.Current(r.Context(), claims)
	if err != nil {
		userapi.WriteCurrentError(w, err)
		return
	}
	userID := current.ID
	adminScopes := strings.TrimSpace(scopeSet) == infiniteflight.ScopeSetAdmin
	vaID := strings.TrimSpace(claims.PbServerID)
	if adminScopes && vaID == "" {
		response.WriteError(w, http.StatusBadRequest, "VA_CONTEXT_REQUIRED", "Virtual airline context is required for admin Live scopes")
		return
	}
	if adminScopes {
		slog.Info("start infinite flight oauth", "user_id", userID, "va_id", vaID, "admin_scopes", true)
	}
	scopes := infiniteflight.DefaultOAuthScopes
	if adminScopes {
		scopes = infiniteflight.AdminOAuthScopes
	}
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
		VaID:         vaID,
		AdminScopes:  adminScopes,
	}
	if err := h.cache.SetJSON(r.Context(), cache.KeyIFOAuthPending(state), pending, infiniteflight.OAuthPendingTTL); err != nil {
		slog.Error("store oauth pending session", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start OAuth")
		return
	}
	authorizeURL, err := h.client.AuthorizationURL(h.redirectURI, state, challenge, scopes)
	if err != nil {
		slog.Error("build oauth authorize url", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start OAuth")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"url": authorizeURL},
	})
}

// CompleteCallback exchanges the authorization code and stores tokens.
func (h *Handler) CompleteCallback(w http.ResponseWriter, r *http.Request, code, state string) {
	if h.client == nil {
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	state = strings.TrimSpace(state)
	code = strings.TrimSpace(code)
	if state == "" || code == "" {
		slog.Info("oauth callback missing parameters")
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	var pending infiniteflight.OAuthPending
	if err := h.cache.GetDelJSON(r.Context(), cache.KeyIFOAuthPending(state), &pending); err != nil {
		if !errors.Is(err, cache.ErrMiss) {
			slog.Error("load oauth pending session", "error", err)
		} else {
			slog.Info("oauth callback unknown or expired state")
		}
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	userID := strings.TrimSpace(pending.UserID)
	if userID == "" {
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	if sessionUserID := strings.TrimSpace(claims.PbUserID); sessionUserID != "" && sessionUserID != userID {
		slog.Info("oauth callback user mismatch")
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	tokens, err := h.client.ExchangeCode(r.Context(), h.redirectURI, code, pending.CodeVerifier)
	if err != nil {
		slog.Error("exchange oauth code", "error", err)
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	if pending.AdminScopes {
		vaID := strings.TrimSpace(pending.VaID)
		if vaID == "" || h.tokens == nil || h.live == nil || h.connections == nil {
			http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
			return
		}
		if err := h.tokens.StoreVATokens(r.Context(), vaID, userID, tokens); err != nil {
			slog.Error("store va oauth tokens", "error", err)
			http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
			return
		}
		orgs, err := h.live.ListOrganizations(r.Context(), tokens.AccessToken)
		if err != nil {
			slog.Error("list organizations after va oauth", "error", err)
			http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
			return
		}
		linkResult, err := valive.LinkAdminOrganizations(r.Context(), h.connections, vaID, userID, orgs)
		if err != nil {
			slog.Error("link admin organizations", "error", err)
			http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
			return
		}
		slog.Info("infinite flight va oauth connected",
			"user_id", userID,
			"va_id", vaID,
			"orgs_linked", linkResult.Linked,
			"orgs_skipped_occupied", linkResult.SkippedOccupied,
		)
		http.Redirect(w, r, routes.AdminLive+"?oauth=connected", http.StatusSeeOther)
		return
	}
	expiresAt := valive.AccessExpiresAt(tokens)
	stored := infiniteflight.OAuthTokenSession{
		AccessToken:     tokens.AccessToken,
		RefreshToken:    tokens.RefreshToken,
		TokenType:       tokens.TokenType,
		Scope:           tokens.Scope,
		AccessExpiresAt: expiresAt,
	}
	// TODO: encrypt OAuth tokens at rest in Redis before a dedicated encryption feature ships.
	if err := h.cache.SetJSON(r.Context(), cache.KeyIFOAuthUser(userID), stored, session.TTL); err != nil {
		slog.Error("store oauth token session", "error", err)
		http.Redirect(w, r, routes.AdminLive+"?oauth=error", http.StatusSeeOther)
		return
	}
	slog.Info("infinite flight user oauth connected", "user_id", userID, "token_store", "user")
	http.Redirect(w, r, routes.AdminLive+"?oauth=connected", http.StatusSeeOther)
}
