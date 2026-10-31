package ui

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	domainflights "infinite-experiment/politburo/internal/livegame/flights"
)

func (h *Handler) MapActiveFlightsPartial(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, mapsActivePageURL(r), http.StatusSeeOther)
		return
	}
	filters, server, _, err := mapFiltersFromRequest(r)
	if err != nil {
		h.renderMapsFlightsPartial(w, r, flightsPartialView(nil, time.Time{}, false, 0, "Invalid filters"))
		return
	}
	if server == "" {
		h.renderMapsFlightsPartial(w, r, flightsPartialView(nil, time.Time{}, false, 0, "Select a server"))
		return
	}
	if h.flights == nil {
		h.renderMapsFlightsPartial(w, r, flightsPartialView(nil, time.Time{}, false, 0, "Flights unavailable"))
		return
	}
	meta, tracks, err := listAllTracks(r, h.flights, server, filters)
	if err != nil {
		msg := "Failed to load flights"
		if errors.Is(err, domainflights.ErrCacheMiss) {
			w.WriteHeader(http.StatusServiceUnavailable)
			msg = "Active flights cache is unavailable"
		} else if errors.Is(err, domainflights.ErrInvalidFilter) {
			w.WriteHeader(http.StatusBadRequest)
			msg = "Invalid filters"
		} else {
			slog.Warn("map active flights partial", "error", err, "serverId", server)
		}
		h.renderMapsFlightsPartial(w, r, flightsPartialView(nil, time.Time{}, false, 0, msg))
		return
	}
	h.renderMapsFlightsPartial(w, r, flightsPartialView(tracks, meta.LastCached, true, meta.TotalLength, ""))
}

func (h *Handler) MapFlightDetailPartial(w http.ResponseWriter, r *http.Request) {
	flightID := strings.TrimSpace(chi.URLParam(r, "flightID"))
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	if tab == "" {
		tab = "flight"
	}
	switch tab {
	case "flight", "route", "user":
	default:
		tab = "flight"
	}

	view := mapsFlightDetailView{FlightID: flightID, Tab: tab}
	if tab != "flight" {
		if err := h.renderer.Render(w, "maps-flight-detail", view); err != nil {
			slog.Error("render map flight detail", "error", err)
		}
		return
	}
	if h.flights == nil || flightID == "" {
		view.GoneMessage = "This flight is no longer active."
		w.WriteHeader(http.StatusNotFound)
		if err := h.renderer.Render(w, "maps-flight-detail", view); err != nil {
			slog.Error("render map flight detail", "error", err)
		}
		return
	}
	record, err := h.flights.GetActiveFlight(r.Context(), flightID)
	if err != nil {
		view.GoneMessage = "This flight is no longer active."
		if errors.Is(err, domainflights.ErrCacheMiss) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			slog.Warn("map flight detail", "error", err, "flightId", flightID)
			w.WriteHeader(http.StatusServiceUnavailable)
			view.GoneMessage = "Failed to load flight details."
		}
		if err := h.renderer.Render(w, "maps-flight-detail", view); err != nil {
			slog.Error("render map flight detail", "error", err)
		}
		return
	}
	rec := recordViewFromDomain(record)
	view.Record = &rec
	if err := h.renderer.Render(w, "maps-flight-detail", view); err != nil {
		slog.Error("render map flight detail", "error", err)
	}
}

func (h *Handler) renderMapsFlightsPartial(w http.ResponseWriter, r *http.Request, data mapsFlightsPartialView) {
	w.Header().Set("HX-Replace-Url", mapsActivePageURL(r))
	if err := h.renderer.Render(w, "maps-flights-status", data); err != nil {
		slog.Error("render map flights status", "error", err)
	}
}
