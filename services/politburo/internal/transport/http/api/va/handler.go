package va

import (
	"context"
	"log/slog"
	"net/http"

	"infinite-experiment/politburo/internal/access/auth"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type vaLoader interface {
	GetByID(ctx context.Context, id string) (*virtualairlines.VirtualAirline, error)
}

type Handler struct {
	vas             vaLoader
	configs         configBlocksRepo
	users           discordUserLookup
	webhookTargets  webhookTargetLoader
	webhookDispatch webhookDispatcher
}

func NewHandler(
	vas *virtualairlines.Repository,
	configs *vaconfig.Repository,
	users *users.Repository,
	vaInit *virtualairlines.Service,
	webhookDispatch webhookDispatcher,
) *Handler {
	var targets webhookTargetLoader
	if vaInit != nil {
		targets = vaInit
	}
	return &Handler{
		vas:             vas,
		configs:         configs,
		users:           users,
		webhookTargets:  targets,
		webhookDispatch: webhookDispatch,
	}
}

func (h *Handler) GetVA(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	vaID := claims.PbServerID
	if vaID == "" {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return
	}

	va, err := h.vas.GetByID(r.Context(), vaID)
	if err != nil {
		slog.Error("load virtual airline", "va_id", vaID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load virtual airline")
		return
	}
	if va == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":       va.ID,
			"name":     va.Name,
			"code":     va.Code,
			"isActive": va.IsActive,
		},
	})
}
