package va

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/membership"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ListVAMembers returns active or removed VA members for staff.
func (h *Handler) ListVAMembers(w http.ResponseWriter, r *http.Request, params politburoapi.ListVAMembersParams) {
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

	active := true
	if params.Status != nil && *params.Status == politburoapi.ListVAMembersParamsStatusDeleted {
		active = false
	}

	members, err := h.memberships.ListForVA(r.Context(), vaID, active)
	if err != nil {
		slog.Error("list va members", "va_id", vaID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "LOAD_FAILED", "failed to load members")
		return
	}

	data := make([]map[string]any, 0, len(members))
	for _, m := range members {
		row := map[string]any{
			"userId":    m.UserID,
			"role":      m.Role,
			"joinedAt":  m.JoinedAt.UTC().Format(timeRFC3339),
			"updatedAt": m.UpdatedAt.UTC().Format(timeRFC3339),
		}
		if m.IFCommunityID != nil {
			row["ifCommunityId"] = *m.IFCommunityID
		}
		if m.Username != nil {
			row["username"] = *m.Username
		}
		if m.Callsign != nil {
			row["callsign"] = *m.Callsign
		}
		data = append(data, row)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": data})
}

// UpdateVAMemberRole changes a member role (administrator only).
func (h *Handler) UpdateVAMemberRole(w http.ResponseWriter, r *http.Request, userID openapi_types.UUID) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	if !claims.IsAdmin() {
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "administrator required")
		return
	}
	vaID := claims.PbServerID
	if vaID == "" {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return
	}
	actorID := claims.PbUserID
	if actorID == "" {
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "user context required")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid body")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}
	role := req.Role
	if !membership.IsMemberRole(role) {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid role")
		return
	}

	err = h.memberships.UpdateRole(r.Context(), membership.UpdateRoleInput{
		ActorUserID:  actorID,
		TargetUserID: userID.String(),
		VAID:         vaID,
		NewRole:      role,
	})
	switch {
	case errors.Is(err, membership.ErrMemberNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
	case errors.Is(err, membership.ErrInvalidRole), errors.Is(err, membership.ErrRoleChangeDenied):
		response.WriteError(w, http.StatusConflict, "ROLE_CHANGE_DENIED", err.Error())
	case err != nil:
		slog.Error("update va member role", "va_id", vaID, "user_id", userID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "UPDATE_FAILED", "failed to update role")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// RemoveVAMember deactivates a VA membership.
func (h *Handler) RemoveVAMember(w http.ResponseWriter, r *http.Request, userID openapi_types.UUID) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	if !claims.IsStaff() {
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "staff required")
		return
	}
	vaID := claims.PbServerID
	if vaID == "" {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "virtual airline not found")
		return
	}
	actorID := claims.PbUserID
	if actorID == "" {
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "user context required")
		return
	}

	err := h.memberships.RemoveMember(r.Context(), membership.RemoveMemberInput{
		ActorUserID:  actorID,
		ActorRole:    claims.Role,
		TargetUserID: userID.String(),
		VAID:         vaID,
	})
	switch {
	case errors.Is(err, membership.ErrMemberNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
	case errors.Is(err, membership.ErrCannotRemoveSelf), errors.Is(err, membership.ErrCannotRemoveAdmin), errors.Is(err, membership.ErrRemovalDenied):
		response.WriteError(w, http.StatusConflict, "REMOVAL_DENIED", err.Error())
	case err != nil:
		slog.Error("remove va member", "va_id", vaID, "user_id", userID, "error", err)
		response.WriteError(w, http.StatusInternalServerError, "REMOVE_FAILED", "failed to remove member")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"
