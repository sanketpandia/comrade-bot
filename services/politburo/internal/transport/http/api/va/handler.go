package va

import (
	"context"
	"log/slog"
	"net/http"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type vaLoader interface {
	GetByID(ctx context.Context, id string) (*virtualairlines.VirtualAirline, error)
}

type Handler struct {
	vas      vaLoader
	configs  configBlocksRepo
	users    discordUserLookup
	webhooks webhookTrigger
}

func NewHandler(
	vas *virtualairlines.Repository,
	configs *vaconfig.Repository,
	users *users.Repository,
	webhooks webhookTrigger,
) *Handler {
	return &Handler{vas: vas, configs: configs, users: users, webhooks: webhooks}
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
