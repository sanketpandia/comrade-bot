package live

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type linkedOrgAccess struct {
	vaID        string
	accessToken string
}

func (h *ConnectionsHandler) resolveLinkedOrgAccess(
	w http.ResponseWriter,
	r *http.Request,
	organizationID string,
) (linkedOrgAccess, bool) {
	var out linkedOrgAccess
	vaID, ok := vaIDFromClaims(w, r)
	if !ok {
		return out, false
	}
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "organization id is required")
		return out, false
	}
	if h.connections == nil || h.tokens == nil {
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "live connections are not configured")
		return out, false
	}
	if h.liveData == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return out, false
	}
	conn, err := h.connections.GetByVAAndOrg(r.Context(), vaID, organizationID)
	if err != nil {
		slog.Error("load va live connection", "va_id", vaID, "organization_id", organizationID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load connection")
		return out, false
	}
	if conn == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "connection not found")
		return out, false
	}
	accessToken, err := h.tokens.AccessToken(r.Context(), vaID)
	if errors.Is(err, valive.ErrOAuthRequired) {
		slog.Info("va live oauth grant missing", "va_id", vaID)
		response.WriteError(w, http.StatusConflict, "IF_OAUTH_REQUIRED", "Connect Infinite Flight in Admin Live Mode first")
		return out, false
	}
	if err != nil {
		slog.Error("resolve va live access token", "va_id", vaID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load OAuth session")
		return out, false
	}
	out.vaID = vaID
	out.accessToken = accessToken
	return out, true
}

func writeLiveUpstreamError(w http.ResponseWriter, err error, logAction string, organizationID string) {
	if err == nil {
		return
	}
	slog.Error(logAction, "organization_id", organizationID, "error", err)
	response.WriteError(w, http.StatusBadGateway, "IF_LIVE_UPSTREAM", "Infinite Flight Live API request failed")
}

func intParam(value *int, defaultValue int) int {
	if value == nil || *value < 1 {
		return defaultValue
	}
	return *value
}
