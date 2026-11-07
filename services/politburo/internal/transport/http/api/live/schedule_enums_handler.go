package live

import (
	"net/http"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/transport/http/response"
)

// ListVALiveScheduleEnums returns valid schedule status and flight type API enum names.
func (h *ConnectionsHandler) ListVALiveScheduleEnums(w http.ResponseWriter, r *http.Request) {
	opts := infiniteflight.ScheduleEnumOptions()
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"status":     opts.Status,
			"flightType": opts.FlightType,
			"crewRole":   opts.CrewRole,
		},
	})
}
