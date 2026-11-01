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
	liveSchedulesPageSize = 100
	liveSchedulesMaxPages = 10
)

func parseQueryAircraftIDs(r *http.Request) []string {
	raw := r.URL.Query()["aircraftId"]
	if len(raw) == 0 {
		if single := strings.TrimSpace(r.URL.Query().Get("aircraftId")); single != "" {
			return []string{single}
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func (h *Handler) LiveSchedulesListPartial(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !claims.IsStaff() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	orgID := strings.TrimSpace(r.URL.Query().Get("organizationId"))
	filterAircraftID := strings.TrimSpace(r.URL.Query().Get("aircraftId"))
	queryAircraftIDs := parseQueryAircraftIDs(r)
	view := liveSchedulesView{OrganizationID: orgID, FilterAircraftID: filterAircraftID}

	if orgID == "" {
		view.ErrorMessage = "Select an organisation."
		h.renderLiveSchedules(w, view)
		return
	}
	if h.liveFleet == nil || h.liveFleet.Live == nil || h.liveFleet.Tokens == nil || h.liveFleet.Connections == nil {
		view.ErrorMessage = "Infinite Flight Live API is not configured."
		h.renderLiveSchedules(w, view)
		return
	}
	vaID := strings.TrimSpace(claims.PbServerID)
	if vaID == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	conn, err := h.liveFleet.Connections.GetByVAAndOrg(r.Context(), vaID, orgID)
	if err != nil {
		slog.Error("load va live connection for schedules", "va_id", vaID, "organization_id", orgID, "error", err)
		view.ErrorMessage = "Could not verify organisation link."
		h.renderLiveSchedules(w, view)
		return
	}
	if conn == nil {
		view.ErrorMessage = "Organisation not linked to this virtual airline."
		h.renderLiveSchedules(w, view)
		return
	}
	accessToken, err := h.liveFleet.Tokens.AccessToken(r.Context(), vaID)
	if errors.Is(err, valive.ErrOAuthRequired) {
		view.ErrorMessage = "Connect Infinite Flight under Admin → Live Mode, then reload this page."
		h.renderLiveSchedules(w, view)
		return
	}
	if err != nil {
		slog.Error("resolve va live access token for schedules", "va_id", vaID, "error", err)
		view.ErrorMessage = "Could not load OAuth session."
		h.renderLiveSchedules(w, view)
		return
	}

	upstream, err := fetchLiveSchedulesUpstream(r.Context(), h.liveFleet.Live, accessToken, orgID, filterAircraftID, queryAircraftIDs)
	if err != nil {
		slog.Error("fetch live schedules upstream", "organization_id", orgID, "error", err)
		if strings.Contains(err.Error(), "list organization aircraft") {
			view.ErrorMessage = "Could not load aircraft from Infinite Flight."
		} else {
			view.ErrorMessage = "Could not load schedules from Infinite Flight."
		}
		h.renderLiveSchedules(w, view)
		return
	}

	var lookup *gameliveries.Lookup
	if h.liveFleet.Liveries != nil {
		lookup = h.liveFleet.Liveries
	}
	view = liveSchedulesViewFromUpstream(orgID, filterAircraftID, upstream.Aircraft, upstream.Schedules, upstream.Truncated, lookup)
	h.renderLiveSchedules(w, view)
}

func (h *Handler) renderLiveSchedules(w http.ResponseWriter, view liveSchedulesView) {
	if err := h.renderer.Render(w, "live-schedules", view); err != nil {
		slog.Error("render live schedules", "error", err)
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}
