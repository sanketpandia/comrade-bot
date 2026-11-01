package ui

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/valive"
)

// LiveFleetDeps wires Live Mode fleet UI to upstream Live API and VA OAuth.
type LiveFleetDeps struct {
	Live        liveFleetAircraftLister
	Tokens      liveFleetTokenSource
	Connections liveFleetConnectionStore
	Liveries    *gameliveries.Lookup
}

type liveFleetConnectionStore interface {
	GetByVAAndOrg(context.Context, string, string) (*valive.Connection, error)
}

type liveFleetTokenSource interface {
	AccessToken(context.Context, string) (string, error)
}

type liveFleetAircraftLister interface {
	liveSchedulesUpstreamClient
	liveFlightsUpstreamClient
}

func (h *Handler) LiveAircraftFleetPartial(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !claims.IsStaff() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	orgID := strings.TrimSpace(r.URL.Query().Get("organizationId"))
	view := liveAircraftFleetView{OrganizationID: orgID}

	if orgID == "" {
		view.ErrorMessage = "Select an organisation."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	if h.liveFleet == nil || h.liveFleet.Live == nil || h.liveFleet.Tokens == nil || h.liveFleet.Connections == nil {
		view.ErrorMessage = "Infinite Flight Live API is not configured."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	vaID := strings.TrimSpace(claims.PbServerID)
	if vaID == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	conn, err := h.liveFleet.Connections.GetByVAAndOrg(r.Context(), vaID, orgID)
	if err != nil {
		slog.Error("load va live connection for fleet", "va_id", vaID, "organization_id", orgID, "error", err)
		view.ErrorMessage = "Could not verify organisation link."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	if conn == nil {
		view.ErrorMessage = "Organisation not linked to this virtual airline."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	accessToken, err := h.liveFleet.Tokens.AccessToken(r.Context(), vaID)
	if errors.Is(err, valive.ErrOAuthRequired) {
		view.ErrorMessage = "Connect Infinite Flight under Admin → Live Mode, then reload this page."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	if err != nil {
		slog.Error("resolve va live access token for fleet", "va_id", vaID, "error", err)
		view.ErrorMessage = "Could not load OAuth session."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	rows, err := h.liveFleet.Live.ListOrganizationAircraft(r.Context(), accessToken, orgID)
	if err != nil {
		slog.Error("list infinite flight organization aircraft", "organization_id", orgID, "error", err)
		view.ErrorMessage = "Could not load aircraft from Infinite Flight."
		h.renderLiveAircraftFleet(w, view)
		return
	}
	var lookup *gameliveries.Lookup
	if h.liveFleet.Liveries != nil {
		lookup = h.liveFleet.Liveries
	}
	view = liveAircraftFleetViewFromUpstream(orgID, rows, lookup)
	h.renderLiveAircraftFleet(w, view)
}

func (h *Handler) renderLiveAircraftFleet(w http.ResponseWriter, view liveAircraftFleetView) {
	if err := h.renderer.Render(w, "live-aircraft-fleet", view); err != nil {
		slog.Error("render live aircraft fleet", "error", err)
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}
