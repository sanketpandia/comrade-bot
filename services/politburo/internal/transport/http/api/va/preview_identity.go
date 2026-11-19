package va

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/community/virtualairlines/featureconfig"
)

func (h *Handler) applyStatsPreviewIdentityDefaults(ctx context.Context, r *http.Request, vaID string, in *featureconfig.PreviewInput) {
	if in == nil || in.Stats == nil {
		return
	}
	needsCallsign, needsIFC := statsPreviewIdentityNeeds(in.Stats)
	beforeCallsign := strings.TrimSpace(in.Callsign)
	beforeIFC := strings.TrimSpace(in.IFCUsername)

	user := h.loadRequestUser(ctx, r)
	if user == nil {
		return
	}
	if needsIFC && beforeIFC == "" && user.Username != nil {
		if name := strings.TrimSpace(*user.Username); name != "" {
			in.IFCUsername = name
		}
	}
	if needsCallsign && beforeCallsign == "" && h.memberships != nil {
		m, err := h.memberships.ActiveMembership(ctx, user.ID, vaID)
		if err == nil && m != nil && m.Callsign != nil {
			if cs := strings.TrimSpace(*m.Callsign); cs != "" {
				in.Callsign = cs
			}
		}
	}
	afterCallsign := strings.TrimSpace(in.Callsign)
	afterIFC := strings.TrimSpace(in.IFCUsername)
	if afterCallsign == beforeCallsign && afterIFC == beforeIFC {
		return
	}
	slog.Info("stats preview identity filled from session",
		"component", "feature_mappings",
		"va_id", vaID,
		"preview_has_callsign", afterCallsign != "",
		"preview_has_ifc_username", afterIFC != "",
	)
}

func statsPreviewIdentityNeeds(doc *featureconfig.StatsDocument) (callsign, ifc bool) {
	if doc.Pilot != nil {
		switch doc.Pilot.MatchBy {
		case "callsign":
			callsign = true
		case "ifcUsername":
			ifc = true
		}
	}
	if doc.Career != nil {
		switch doc.Career.MatchBy {
		case "callsign":
			callsign = true
		case "ifcUsername":
			ifc = true
		}
	}
	return callsign, ifc
}

func (h *Handler) loadRequestUser(ctx context.Context, r *http.Request) *users.User {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || h.users == nil {
		return nil
	}
	if id := strings.TrimSpace(claims.PbUserID); id != "" {
		user, err := h.users.GetByID(ctx, id)
		if err == nil && user != nil {
			return user
		}
	}
	if ds := strings.TrimSpace(claims.DsUserID); ds != "" {
		user, err := h.users.GetByDiscordID(ctx, ds)
		if err == nil && user != nil {
			return user
		}
	}
	return nil
}