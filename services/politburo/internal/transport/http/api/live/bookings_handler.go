package live

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"infinite-experiment/politburo/internal/transport/http/response"
)

func (h *ConnectionsHandler) ListVALiveScheduleBookings(w http.ResponseWriter, r *http.Request) {
	vaID, ok := vaIDFromClaims(w, r)
	if !ok {
		return
	}
	if h.actions == nil {
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "live action history is not configured")
		return
	}
	var liveOrgFilter *string
	if raw := strings.TrimSpace(r.URL.Query().Get("organizationId")); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			response.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "organizationId must be a UUID")
			return
		}
		liveOrgFilter = &raw
	}
	rows, err := h.actions.ListCrewAssignments(r.Context(), vaID, liveOrgFilter)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load booking history")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{
			"id":        row.ID,
			"startedAt": row.StartedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"outcome":   row.Outcome,
		}
		if row.UpstreamStatus != nil {
			item["upstreamStatus"] = *row.UpstreamStatus
		}
		if row.ErrorMessage != nil {
			item["errorMessage"] = *row.ErrorMessage
		}
		if row.LiveOrganizationID != nil {
			item["organizationId"] = *row.LiveOrganizationID
		}
		if row.ActorUsername != nil {
			item["actorUsername"] = *row.ActorUsername
		}
		if row.ActorIFCUsername != nil {
			item["actorIfcUsername"] = *row.ActorIFCUsername
		}
		appendMetaFields(item, row.Meta)
		out = append(out, item)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func appendMetaFields(item map[string]any, meta map[string]any) {
	if meta == nil {
		return
	}
	for _, key := range []string{"registration", "route", "callsign", "originIcao", "destinationIcao", "scheduleId"} {
		if v, ok := meta[key]; ok {
			s := strings.TrimSpace(anyString(v))
			if s != "" {
				item[key] = s
			}
		}
	}
}

func anyString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
