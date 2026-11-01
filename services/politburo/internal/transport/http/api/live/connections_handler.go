package live

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/livegame/valive"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type connectionsRepo interface {
	ListByVAID(ctx context.Context, vaID string) ([]valive.Connection, error)
	GetByVAAndOrg(ctx context.Context, vaID, orgID string) (*valive.Connection, error)
	Delete(ctx context.Context, vaID, orgID, actorUserID string) error
}

type liveOrgClient interface {
	GetOrganization(context.Context, string, string) (infiniteflight.LiveOrganization, error)
}

type vaTokenProvider interface {
	AccessToken(context.Context, string) (string, error)
}

// ConnectionsHandler serves persisted VA Live organisation connections.
type ConnectionsHandler struct {
	connections connectionsRepo
	live        liveOrgClient
	tokens      vaTokenProvider
	users       currentUser
}

func NewConnectionsHandler(
	connections *valive.ConnectionsRepository,
	live *infiniteflight.LiveClient,
	tokens *valive.TokenProvider,
	users currentUser,
) *ConnectionsHandler {
	return &ConnectionsHandler{connections: connections, live: live, tokens: tokens, users: users}
}

func (h *ConnectionsHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	vaID, ok := vaIDFromClaims(w, r)
	if !ok {
		return
	}
	rows, err := h.connections.ListByVAID(r.Context(), vaID)
	if err != nil {
		slog.Error("list va live connections", "va_id", vaID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load connections")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":       row.LiveOrganizationID,
			"name":     row.OrganizationName,
			"role":     row.OrganizationRole,
			"linkedAt": row.LinkedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	if len(out) == 0 {
		slog.Debug("va live connections empty", "va_id", vaID, "count", 0)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *ConnectionsHandler) DeleteConnection(w http.ResponseWriter, r *http.Request, organizationID string) {
	vaID, ok := vaIDFromClaims(w, r)
	if !ok {
		return
	}
	organizationID = strings.TrimSpace(organizationID)
	claims, _ := auth.ClaimsFromContext(r.Context())
	current, err := h.users.Current(r.Context(), claims)
	if err != nil {
		userapi.WriteCurrentError(w, err)
		return
	}
	err = h.connections.Delete(r.Context(), vaID, organizationID, current.ID)
	if errors.Is(err, sql.ErrNoRows) {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "connection not found")
		return
	}
	if err != nil {
		slog.Error("delete va live connection", "va_id", vaID, "organization_id", organizationID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not remove connection")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConnectionsHandler) PingConnection(w http.ResponseWriter, r *http.Request, organizationID string) {
	vaID, ok := vaIDFromClaims(w, r)
	if !ok {
		return
	}
	organizationID = strings.TrimSpace(organizationID)
	if h.live == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return
	}
	conn, err := h.connections.GetByVAAndOrg(r.Context(), vaID, organizationID)
	if err != nil {
		slog.Error("load va live connection", "va_id", vaID, "organization_id", organizationID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load connection")
		return
	}
	if conn == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "connection not found")
		return
	}
	accessToken, err := h.tokens.AccessToken(r.Context(), vaID)
	if errors.Is(err, valive.ErrOAuthRequired) {
		slog.Info("va live oauth grant missing", "va_id", vaID)
		response.WriteError(w, http.StatusConflict, "IF_OAUTH_REQUIRED", "Connect Infinite Flight in Admin Live Mode first")
		return
	}
	if err != nil {
		slog.Error("resolve va live access token", "va_id", vaID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load OAuth session")
		return
	}
	org, err := h.live.GetOrganization(r.Context(), accessToken, organizationID)
	if errors.Is(err, infiniteflight.ErrLiveOrganizationNotFound) {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "organisation not found upstream")
		return
	}
	if err != nil {
		slog.Error("ping infinite flight organization", "organization_id", organizationID, "error", err)
		response.WriteError(w, http.StatusBadGateway, "IF_LIVE_UPSTREAM", "Infinite Flight Live API request failed")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":   org.ID,
			"name": org.Name,
			"role": org.Role,
		},
	})
}

func vaIDFromClaims(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return "", false
	}
	vaID := strings.TrimSpace(claims.PbServerID)
	if vaID == "" {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return "", false
	}
	return vaID, true
}
