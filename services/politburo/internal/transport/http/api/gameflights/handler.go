package gameflights

import (
	"net/http"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Handler struct {
	svc *Service
}

func NewHandler(reader *domainflights.Reader, metricsRegistry *metrics.Registry) *Handler {
	return &Handler{svc: NewService(reader, metricsRegistry)}
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

	resp, err := h.svc.ListActiveTracks(r.Context(), input)
	if err != nil {
		writeListActiveTracksError(w, input.Server, input.Filters, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, resp)
}
