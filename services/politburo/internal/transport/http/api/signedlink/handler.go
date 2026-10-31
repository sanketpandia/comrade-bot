package signedlink

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type userLookup interface {
	GetByDiscordID(ctx context.Context, discordID string) (*users.User, error)
}

type ticketIssuer interface {
	Issue(ctx context.Context, ticket auth.LoginTicket) (string, error)
}

type membershipResolver interface {
	ResolveMembership(ctx context.Context, discordUserID, discordServerID string) (vaID, role string, ok bool, err error)
}

type vaByDiscordServer interface {
	IsConfiguredVA(ctx context.Context, discordServerID string) (bool, error)
}

type Handler struct {
	svc *Service
}

func NewHandler(users userLookup, tickets ticketIssuer, membership membershipResolver, vaLookup vaByDiscordServer, mapLink vaMapLinkResolver, uiBaseURL string) *Handler {
	return &Handler{svc: NewService(users, tickets, membership, vaLookup, mapLink, uiBaseURL)}
}

type requestBody struct {
	Type        string `json:"type"`
	RedirectTo  string `json:"redirectTo"`
	RedirectTo2 string `json:"redirect_to"`
}

func (h *Handler) GenerateSignedLink(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	if claims.DsUserID == "" {
		response.WriteError(w, http.StatusForbidden, "MISSING_DISCORD_CONTEXT", "Missing required Discord context header: X-Discord-User-Id")
		return
	}

	var body requestBody
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil && err != io.EOF {
			response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
	}

	customRedirect := firstNonEmpty(body.RedirectTo, body.RedirectTo2)
	result, err := h.svc.Generate(r.Context(), claims, body.Type, customRedirect)
	if err != nil {
		writeGenerateError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"url":        result.URL,
			"expiresIn":  result.ExpiresIn,
			"redirectTo": result.RedirectTo,
		},
	})
}

func writeGenerateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMissingLinkType):
		response.WriteError(w, http.StatusBadRequest, "MISSING_LINK_TYPE", "type is required")
	case errors.Is(err, ErrInvalidLinkType):
		response.WriteError(w, http.StatusBadRequest, "INVALID_LINK_TYPE", "type must be DASHBOARD, LIVE_MAP, or CUSTOM")
	case errors.Is(err, ErrMissingRedirect):
		response.WriteError(w, http.StatusBadRequest, "MISSING_REDIRECT", "redirectTo is required when type is CUSTOM")
	case errors.Is(err, ErrInvalidRedirect):
		response.WriteError(w, http.StatusBadRequest, "INVALID_REDIRECT", "redirectTo must be a relative path")
	case errors.Is(err, ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	case errors.Is(err, ErrNotVAMember):
		response.WriteError(w, http.StatusForbidden, "NOT_VA_MEMBER", "You must join this virtual airline before opening the portal")
	case errors.Is(err, ErrUserLookupFailed):
		response.WriteError(w, http.StatusInternalServerError, "USER_LOOKUP_FAILED", "user lookup failed")
	case errors.Is(err, ErrVALookupFailed):
		response.WriteError(w, http.StatusInternalServerError, "VA_LOOKUP_FAILED", "va lookup failed")
	case errors.Is(err, ErrMembershipLookupFailed):
		response.WriteError(w, http.StatusInternalServerError, "MEMBERSHIP_LOOKUP_FAILED", "membership lookup failed")
	case errors.Is(err, ErrMapLinkFailed):
		response.WriteError(w, http.StatusInternalServerError, "GENERATION_FAILED", "failed to generate signed link")
	case errors.Is(err, ErrTicketIssueFailed):
		response.WriteError(w, http.StatusInternalServerError, "GENERATION_FAILED", "failed to generate signed link")
	default:
		response.WriteError(w, http.StatusInternalServerError, "GENERATION_FAILED", "failed to generate signed link")
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}
