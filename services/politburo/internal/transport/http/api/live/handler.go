package live

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
	"infinite-experiment/politburo/internal/transport/http/response"
)

const (
	roleAdmin = 1
	roleOwner = 2
)

type tokenReader interface {
	GetJSON(context.Context, string, any) error
}

type organizationsClient interface {
	ListOrganizations(context.Context, string) ([]infiniteflight.LiveOrganization, error)
}

type currentUser interface {
	Current(context.Context, auth.Claims) (*users.User, error)
}

// Handler serves Politburo Live Mode JSON endpoints backed by Infinite Flight OAuth tokens.
type Handler struct {
	live  organizationsClient
	cache tokenReader
	users currentUser
}

func NewHandler(live organizationsClient, cache tokenReader, users currentUser) *Handler {
	return &Handler{live: live, cache: cache, users: users}
}

type organizationItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role int    `json:"role"`
}

// ListOrganizations returns Live organizations where the caller is Admin or Owner.
func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	if h.live == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	current, err := h.users.Current(r.Context(), claims)
	if err != nil {
		userapi.WriteCurrentError(w, err)
		return
	}
	var token infiniteflight.OAuthTokenSession
	if err := h.cache.GetJSON(r.Context(), cache.KeyIFOAuthUser(current.ID), &token); err != nil {
		if errors.Is(err, cache.ErrMiss) {
			response.WriteError(w, http.StatusConflict, "IF_OAUTH_REQUIRED", "Connect Infinite Flight in Admin Live Mode first")
			return
		}
		slog.Error("load infinite flight oauth token", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load OAuth session")
		return
	}
	if accessTokenExpired(token.AccessExpiresAt) {
		response.WriteError(w, http.StatusConflict, "IF_OAUTH_REQUIRED", "Infinite Flight access token expired; reconnect in Admin Live Mode")
		return
	}
	orgs, err := h.live.ListOrganizations(r.Context(), token.AccessToken)
	if err != nil {
		slog.Error("list infinite flight organizations", "error", err)
		response.WriteError(w, http.StatusBadGateway, "IF_LIVE_UPSTREAM", "Infinite Flight Live API request failed")
		return
	}
	out := make([]organizationItem, 0, len(orgs))
	for _, org := range orgs {
		if org.Role != roleAdmin && org.Role != roleOwner {
			continue
		}
		out = append(out, organizationItem{
			ID:   org.ID,
			Name: org.Name,
			Role: org.Role,
		})
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func accessTokenExpired(expiresAt time.Time) bool {
	return !expiresAt.IsZero() && !time.Now().UTC().Before(expiresAt)
}
