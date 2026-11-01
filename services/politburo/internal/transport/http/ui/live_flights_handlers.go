package ui

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/valive"
)

const (
	liveFlightsPageSize = 100
	liveFlightsMaxPages = 10
)

func (h *Handler) LiveFlightsListPartial(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !claims.IsStaff() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	orgID := strings.TrimSpace(r.URL.Query().Get("organizationId"))
	filterAircraftID := strings.TrimSpace(r.URL.Query().Get("aircraftId"))
	view := liveFlightsView{OrganizationID: orgID, FilterAircraftID: filterAircraftID}

	if orgID == "" {
		view.ErrorMessage = "Select an organisation."
		h.renderLiveFlights(w, view)
		return
	}
	if h.liveFleet == nil || h.liveFleet.Live == nil || h.liveFleet.Tokens == nil || h.liveFleet.Connections == nil {
		view.ErrorMessage = "Infinite Flight Live API is not configured."
		h.renderLiveFlights(w, view)
		return
	}
	vaID := strings.TrimSpace(claims.PbServerID)
	if vaID == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	conn, err := h.liveFleet.Connections.GetByVAAndOrg(r.Context(), vaID, orgID)
	if err != nil {
		slog.Error("load va live connection for flights", "va_id", vaID, "organization_id", orgID, "error", err)
		view.ErrorMessage = "Could not verify organisation link."
		h.renderLiveFlights(w, view)
		return
	}
	if conn == nil {
		view.ErrorMessage = "Organisation not linked to this virtual airline."
		h.renderLiveFlights(w, view)
		return
	}
	accessToken, err := h.liveFleet.Tokens.AccessToken(r.Context(), vaID)
	if errors.Is(err, valive.ErrOAuthRequired) {
		view.ErrorMessage = "Connect Infinite Flight under Admin → Live Mode, then reload this page."
		h.renderLiveFlights(w, view)
		return
	}
	if err != nil {
		slog.Error("resolve va live access token for flights", "va_id", vaID, "error", err)
		view.ErrorMessage = "Could not load OAuth session."
		h.renderLiveFlights(w, view)
		return
	}

	upstream, err := fetchLiveFlightsUpstream(r.Context(), h.liveFleet.Live, accessToken, orgID, filterAircraftID)
	if err != nil {
		slog.Error("fetch live flights upstream", "organization_id", orgID, "error", err)
		if strings.Contains(err.Error(), "list organization aircraft") {
			view.ErrorMessage = "Could not load aircraft from Infinite Flight."
		} else {
			view.ErrorMessage = "Could not load flights from Infinite Flight."
		}
		h.renderLiveFlights(w, view)
		return
	}

	var lookup *gameliveries.Lookup
	if h.liveFleet.Liveries != nil {
		lookup = h.liveFleet.Liveries
	}
	view = liveFlightsViewFromUpstream(orgID, filterAircraftID, upstream.Aircraft, upstream.Flights, upstream.Truncated, lookup)
	h.renderLiveFlights(w, view)
}

func (h *Handler) renderLiveFlights(w http.ResponseWriter, view liveFlightsView) {
	if err := h.renderer.Render(w, "live-flights", view); err != nil {
		slog.Error("render live flights", "error", err)
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}
