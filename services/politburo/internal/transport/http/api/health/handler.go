package health

import (
	"context"
	"net/http"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/transport/http/response"
)

const (
	serviceDatabase = "database"
	serviceCache    = "cache"
	serviceSessions = "infinite-flight"
	stateActive     = "active"
	stateDown       = "down"
)

type Handler struct {
	db              dbPinger
	cache           cacheReader
	monitorSessions bool
	startedAt       time.Time
	now             func() time.Time
}

type dbPinger interface {
	PingContext(context.Context) error
}

type cacheReader interface {
	Ping(context.Context) error
	GetJSON(context.Context, string, any) error
}

func NewHandler(db dbPinger, cacheStore cacheReader, monitorSessions bool, startedAt time.Time) *Handler {
	return &Handler{
		db:              db,
		cache:           cacheStore,
		monitorSessions: monitorSessions,
		startedAt:       startedAt,
		now:             time.Now,
	}
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	services := map[string]string{
		serviceDatabase: stateDown,
		serviceCache:    stateDown,
		serviceSessions: stateDown,
	}
	body := map[string]any{
		"started_at": h.startedAt,
		"uptime":     time.Since(h.startedAt).Round(time.Second).String(),
		"services":   services,
	}

	if err := h.db.PingContext(ctx); err != nil {
		body["status"] = stateDown
		response.WriteJSON(w, http.StatusServiceUnavailable, body)
		return
	}
	services[serviceDatabase] = stateActive

	if err := h.cache.Ping(ctx); err != nil {
		body["status"] = stateDown
		response.WriteJSON(w, http.StatusServiceUnavailable, body)
		return
	}
	services[serviceCache] = stateActive

	if !h.monitorSessions || h.sessionsActive(ctx) {
		services[serviceSessions] = stateActive
	} else {
		body["status"] = stateDown
		response.WriteJSON(w, http.StatusServiceUnavailable, body)
		return
	}

	body["status"] = "ok"
	response.WriteJSON(w, http.StatusOK, body)
}

func (h *Handler) sessionsActive(ctx context.Context) bool {
	if !h.monitorSessions {
		return true
	}

	snapshot := gamesessions.Snapshot{}
	if err := h.cache.GetJSON(ctx, cache.KeyActiveSessions, &snapshot); err != nil {
		return false
	}
	return gamesessions.SnapshotFresh(snapshot.LastCached, h.now().UTC())
}
