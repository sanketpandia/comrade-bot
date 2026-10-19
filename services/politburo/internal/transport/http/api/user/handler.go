package user

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type userLoader interface {
	GetByID(ctx context.Context, id string) (*users.User, error)
	GetByDiscordID(ctx context.Context, discordID string) (*users.User, error)
}

type Handler struct {
	users userLoader
}

func NewHandler(repo *users.Repository) *Handler {
	return &Handler{users: repo}
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var (
		user *users.User
		err  error
	)
	switch {
	case claims.PbUserID != "":
		user, err = h.users.GetByID(r.Context(), claims.PbUserID)
	case strings.TrimSpace(claims.DsUserID) != "":
		user, err = h.users.GetByDiscordID(r.Context(), strings.TrimSpace(claims.DsUserID))
	default:
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	if err != nil {
		slog.Error("load current user", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load user")
		return
	}
	if user == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	data := map[string]any{
		"id":        user.ID,
		"discordId": user.DiscordID,
		"isActive":  user.IsActive,
		"role":      roleJSON(claims.Role),
	}
	if user.Username != nil && strings.TrimSpace(*user.Username) != "" {
		data["username"] = strings.TrimSpace(*user.Username)
	}
	if user.IFCommunityID != nil && strings.TrimSpace(*user.IFCommunityID) != "" {
		data["ifCommunityId"] = strings.TrimSpace(*user.IFCommunityID)
	}
	if claims.PbServerID != "" {
		data["vaId"] = claims.PbServerID
	}
	if strings.TrimSpace(claims.DsServerID) != "" {
		data["discordServerId"] = strings.TrimSpace(claims.DsServerID)
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{"data": data})
}

func roleJSON(role string) any {
	if strings.TrimSpace(role) == "" {
		return nil
	}
	return role
}
