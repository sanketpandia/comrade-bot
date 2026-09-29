package signedlink

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/auth"
	"infinite-experiment/politburo/internal/transport/http/response"
	"infinite-experiment/politburo/internal/users"
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
	users       userLookup
	tickets     ticketIssuer
	membership  membershipResolver
	vaLookup    vaByDiscordServer
	uiBaseURL   string
}

func NewHandler(users userLookup, tickets ticketIssuer, membership membershipResolver, vaLookup vaByDiscordServer, uiBaseURL string) *Handler {
	return &Handler{users: users, tickets: tickets, membership: membership, vaLookup: vaLookup, uiBaseURL: uiBaseURL}
}

type requestBody struct {
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
	redirectTo, err := auth.NormalizeRedirect(firstNonEmpty(body.RedirectTo, body.RedirectTo2))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REDIRECT", "redirectTo must be a relative path")
		return
	}

	user, err := h.users.GetByDiscordID(r.Context(), claims.DsUserID)
	if err != nil {
		slog.Error("lookup user for signed link", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "USER_LOOKUP_FAILED", "user lookup failed")
		return
	}
	if user == nil {
		response.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	vaID, role := "", ""
	if claims.DsServerID != "" && h.vaLookup != nil {
		configured, err := h.vaLookup.IsConfiguredVA(r.Context(), claims.DsServerID)
		if err != nil {
			slog.Error("lookup va for signed link", "error", err)
			response.WriteError(w, http.StatusInternalServerError, "VA_LOOKUP_FAILED", "va lookup failed")
			return
		}
		if configured && h.membership != nil {
			resolvedVA, resolvedRole, ok, err := h.membership.ResolveMembership(r.Context(), claims.DsUserID, claims.DsServerID)
			if err != nil {
				slog.Error("resolve membership for signed link", "error", err)
				response.WriteError(w, http.StatusInternalServerError, "MEMBERSHIP_LOOKUP_FAILED", "membership lookup failed")
				return
			}
			if !ok {
				response.WriteError(w, http.StatusForbidden, "NOT_VA_MEMBER", "You must join this virtual airline before opening the portal")
				return
			}
			vaID, role = resolvedVA, resolvedRole
		}
	}

	token, err := h.tickets.Issue(r.Context(), auth.LoginTicket{
		UserID:          user.ID,
		DiscordUserID:   user.DiscordID,
		DiscordServerID: claims.DsServerID,
		VaID:            vaID,
		Role:            role,
		Username:        user.DisplayName(),
		RedirectTo:      redirectTo,
	})
	if err != nil {
		slog.Error("issue signed link ticket", "error", err)
		response.WriteError(w, http.StatusInternalServerError, "GENERATION_FAILED", "failed to generate signed link")
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"url":        auth.FormatLoginURL(h.uiBaseURL, token),
			"expiresIn":  int(auth.TicketTTL.Seconds()),
			"redirectTo": redirectTo,
		},
	})
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}
