package live

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/community/registration/proof"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/webhooks/features/livemodeassignment"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (h *ConnectionsHandler) AssignVALiveScheduleToMe(
	w http.ResponseWriter,
	r *http.Request,
	organizationID openapi_types.UUID,
) {
	orgID := strings.TrimSpace(organizationID.String())
	vaID, ok := vaIDFromClaims(w, r)
	if !ok {
		return
	}
	if h.assignCaptain == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return
	}
	body, ok := readAssignMeBody(w, r)
	if !ok {
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	user, err := h.users.Current(r.Context(), claims)
	if err != nil {
		userapi.WriteCurrentError(w, err)
		return
	}
	if user.IFApiID == nil || strings.TrimSpace(*user.IFApiID) == "" {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Infinite Flight account not linked; register with /register first")
		return
	}

	result, err := h.assignCaptain.AssignCaptain(r.Context(), valive.AssignCaptainInput{
		VAID:           vaID,
		OrganizationID: orgID,
		ActorUserID:    actorUserID(r),
		IFUserID:       strings.TrimSpace(*user.IFApiID),
		Registration:   body.Registration,
		Route:          body.Route,
	})
	if err != nil {
		writeAssignMeError(w, err, orgID)
		return
	}

	if h.assignmentNotifier != nil {
		notifyErr := h.assignmentNotifier.Notify(r.Context(), vaID, livemodeassignment.Event{
			IFUserID:        strings.TrimSpace(*user.IFApiID),
			PilotUsername:   user.DisplayName(),
			DiscordID:       strings.TrimSpace(user.DiscordID),
			Registration:    result.Registration,
			OriginIcao:      result.OriginIcao,
			DestinationIcao: result.DestinationIcao,
			Callsign:        result.Callsign,
			ScheduleID:      result.ScheduleID,
		})
		if notifyErr != nil {
			slog.Warn("live_mode_assignment webhook failed", "va_id", vaID, "schedule_id", result.ScheduleID, "error", notifyErr)
		}
	}

	data := map[string]any{
		"scheduleId":      result.ScheduleID,
		"organizationId":  result.OrganizationID,
		"registration":    result.Registration,
		"originIcao":      result.OriginIcao,
		"destinationIcao": result.DestinationIcao,
	}
	if result.Callsign != "" {
		data["callsign"] = result.Callsign
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": data})
}

func readAssignMeBody(w http.ResponseWriter, r *http.Request) (politburoapi.LiveScheduleAssignMeBody, bool) {
	var empty politburoapi.LiveScheduleAssignMeBody
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "could not read request body")
		return empty, false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "request body is required")
		return empty, false
	}
	var parsed politburoapi.LiveScheduleAssignMeBody
	if err := json.Unmarshal(raw, &parsed); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "invalid JSON body")
		return empty, false
	}
	if strings.TrimSpace(parsed.Registration) == "" || strings.TrimSpace(parsed.Route) == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "registration and route are required")
		return empty, false
	}
	return parsed, true
}

func writeAssignMeError(w http.ResponseWriter, err error, organizationID string) {
	switch {
	case errors.Is(err, proof.ErrInvalidRoute):
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
	case errors.Is(err, valive.ErrConnectionNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "connection not found")
	case errors.Is(err, valive.ErrRegistrationNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "aircraft registration not found in organisation fleet")
	case errors.Is(err, valive.ErrScheduleNotFound):
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "no open schedule matches that registration and route")
	case errors.Is(err, valive.ErrLiveClientDisabled):
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
	default:
		writeScheduleWriteError(w, err, organizationID)
	}
}
