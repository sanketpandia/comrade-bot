package user

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/transport/http/response"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrNotFound     = errors.New("user not found")
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

// Current loads the caller the same way GET /api/v1/user/me does: Politburo user id
// from a session, otherwise Discord user id from an API-key request.
func (h *Handler) Current(ctx context.Context, claims auth.Claims) (*users.User, error) {
	var (
		user *users.User
		err  error
	)
	switch {
	case claims.PbUserID != "":
		user, err = h.users.GetByID(ctx, claims.PbUserID)
	case strings.TrimSpace(claims.DsUserID) != "":
		user, err = h.users.GetByDiscordID(ctx, strings.TrimSpace(claims.DsUserID))
	default:
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	user, err := h.Current(r.Context(), claims)
	if err != nil {
		WriteCurrentError(w, err)
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

func WriteCurrentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	case errors.Is(err, ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
	default:
		slog.Error("load current user", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load user")
	}
}

func roleJSON(role string) any {
	if strings.TrimSpace(role) == "" {
		return nil
	}
	return role
}
