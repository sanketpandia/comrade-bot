package gameflights

import (
	"errors"
	"net/http"
	"strings"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"log/slog"

	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Handler struct {
	svc      *Service
	vaLookup vaCodeResolver
}

func NewHandler(reader *domainflights.Reader, metricsRegistry *metrics.Registry, vaLookup vaCodeResolver) *Handler {
	return &Handler{svc: NewService(reader, metricsRegistry), vaLookup: vaLookup}
}

func (h *Handler) GetActiveFlight(w http.ResponseWriter, r *http.Request, flightID openapi_types.UUID) {
	resp, err := h.svc.GetActiveFlight(r.Context(), flightID)
	if err != nil {
		writeGetActiveFlightError(w, flightID.String(), err)
		return
	}
	response.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetActiveFlights(w http.ResponseWriter, r *http.Request, normalizedServerName string, params politburoapi.GetActiveFlightsParams) {
	input, err := ListActiveTracksInput(r, normalizedServerName, params)
	if err != nil {
		writeListActiveTracksRequestError(w, err)
		return
	}
	if input.Filters.VACode != "" {
		vaID, err := resolveVACode(r.Context(), h.vaLookup, input.Filters.VACode)
		if err != nil {
			if errors.Is(err, domainflights.ErrUnknownVACode) {
				writeListActiveTracksRequestError(w, err)
				return
			}
			slog.Error("resolve virtual airline code", "error", err, "vaCode", input.Filters.VACode)
			response.WriteError(w, http.StatusInternalServerError, "ACTIVE_FLIGHTS_CACHE_UNAVAILABLE", "active flights cache is unavailable")
			return
		}
		input.Filters.VAID = vaID
		input.Filters.VACode = strings.ToUpper(strings.TrimSpace(input.Filters.VACode))
	}

	resp, err := h.svc.ListActiveTracks(r.Context(), input)
	if err != nil {
		writeListActiveTracksError(w, input.Server, input.Filters, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, resp)
}
