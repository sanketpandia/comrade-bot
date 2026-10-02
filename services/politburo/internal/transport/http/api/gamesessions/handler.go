package gamesessions

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"infinite-experiment/politburo/internal/cache"
	domainsessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/transport/http/api/cachedresponse"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type Handler struct {
	cache cache.Store
}

func NewHandler(cacheStore cache.Store) *Handler {
	return &Handler{cache: cacheStore}
}

// ActiveSession is the public shape for GET /api/v1/game/sessions/active.
type ActiveSession struct {
	NormalizedName string `json:"normalizedName"`
	UserCount      int    `json:"userCount"`
	Type           int    `json:"type"`
}

func (h *Handler) GetActiveSessions(w http.ResponseWriter, r *http.Request) {
	snapshot := domainsessions.Snapshot{}
	if err := h.cache.GetJSON(r.Context(), cache.KeyActiveSessions, &snapshot); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, cache.ErrMiss) {
			status = http.StatusServiceUnavailable
			slog.Warn("active sessions cache miss", "error", err)
		} else {
			slog.Error("read active sessions cache", "error", err)
		}
		response.WriteError(w, status, "ACTIVE_SESSIONS_CACHE_UNAVAILABLE", "active sessions cache is unavailable")
		return
	}
	if snapshot.LastCached.IsZero() {
		slog.Error("read active sessions cache", "error", "lastCached is missing")
		response.WriteError(w, http.StatusInternalServerError, "ACTIVE_SESSIONS_CACHE_UNAVAILABLE", "active sessions cache is unavailable")
		return
	}

	result := make([]ActiveSession, 0, len(snapshot.Result))
	for _, session := range snapshot.Result {
		result = append(result, ActiveSession{
			NormalizedName: session.NormalizedName,
			UserCount:      session.UserCount,
			Type:           session.Type,
		})
	}

	response.WriteJSON(w, http.StatusOK, activeSessionsResponse{
		Data: activeSessionsData{
			Result: result,
			Meta: cachedresponse.Meta{
				LastCached:          snapshot.LastCached,
				RefreshIntervalMins: int(domainsessions.RefreshInterval / time.Minute),
			},
		},
	})
}

type activeSessionsResponse struct {
	Data activeSessionsData `json:"data"`
}

type activeSessionsData struct {
	Result []ActiveSession      `json:"result"`
	Meta   cachedresponse.Meta  `json:"meta"`
}
