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
	reader *domainsessions.Reader
}

func NewHandler(reader *domainsessions.Reader) *Handler {
	return &Handler{reader: reader}
}

type activeSession struct {
	NormalizedName string `json:"normalizedName"`
	UserCount      int    `json:"userCount"`
	Type           int    `json:"type"`
}

func (h *Handler) GetActiveSessions(w http.ResponseWriter, r *http.Request) {
	result, err := h.reader.ListActive(r.Context())
	if err != nil {
		status := http.StatusInternalServerError
		code := "ACTIVE_SESSIONS_CACHE_UNAVAILABLE"
		msg := "active sessions cache is unavailable"
		switch {
		case errors.Is(err, domainsessions.ErrCacheMiss):
			status = http.StatusServiceUnavailable
			slog.Warn("active sessions cache miss", "error", err)
		case errors.Is(err, domainsessions.ErrCacheCorrupt):
			slog.Error("read active sessions cache", "error", "lastCached is missing")
		case errors.Is(err, domainsessions.ErrCacheRead):
			slog.Error("read active sessions cache", "error", err)
		default:
			slog.Error("read active sessions cache", "error", err)
		}
		response.WriteError(w, status, code, msg)
		return
	}

	sessions := make([]activeSession, 0, len(result.Sessions))
	for _, session := range result.Sessions {
		sessions = append(sessions, activeSession{
			NormalizedName: session.NormalizedName,
			UserCount:      session.UserCount,
			Type:           session.Type,
		})
	}

	response.WriteJSON(w, http.StatusOK, activeSessionsResponse{
		Data: activeSessionsData{
			Result: sessions,
			Meta: cachedresponse.Meta{
				LastCached:          result.LastCached,
				RefreshIntervalMins: int(cache.SessionsRefreshInterval / time.Minute),
			},
		},
	})
}

type activeSessionsResponse struct {
	Data activeSessionsData `json:"data"`
}

type activeSessionsData struct {
	Result []activeSession     `json:"result"`
	Meta   cachedresponse.Meta `json:"meta"`
}
