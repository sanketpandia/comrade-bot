package live

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"infinite-experiment/politburo/internal/access/auth"
	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	actionCreateSchedule      = "create_va_live_schedule"
	actionUpdateSchedule      = "update_va_live_schedule"
	actionUpdateScheduleFPL   = "update_va_live_schedule_flightplan"
)

func (h *ConnectionsHandler) CreateVALiveAircraftSchedule(
	w http.ResponseWriter,
	r *http.Request,
	organizationID openapi_types.UUID,
	aircraftID openapi_types.UUID,
) {
	orgID := organizationID.String()
	acID := strings.TrimSpace(aircraftID.String())
	access, ok := h.resolveLinkedOrgAccess(w, r, orgID)
	if !ok {
		return
	}
	if acID == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "aircraft id is required")
		return
	}
	inFleet, err := h.aircraftInOrganizationFleet(r.Context(), access.accessToken, orgID, acID)
	if err != nil {
		writeLiveUpstreamError(w, err, "list infinite flight organization aircraft for schedule create", orgID)
		return
	}
	if !inFleet {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "aircraft not found in organisation fleet")
		return
	}
	body, ok := readScheduleUpsertBody(w, r)
	if !ok {
		return
	}
	if h.liveData == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return
	}
	vaID := access.vaID
	orgCopy := orgID
	actorID := actorUserID(r)
	if err := h.tokens.Write(r.Context(), vaID, actorID, &orgCopy, actionCreateSchedule, nil, func(token string) error {
		return h.liveData.CreateAircraftSchedule(r.Context(), token, acID, body)
	}); err != nil {
		writeScheduleWriteError(w, err, orgID)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConnectionsHandler) UpdateVALiveSchedule(
	w http.ResponseWriter,
	r *http.Request,
	organizationID openapi_types.UUID,
	scheduleID openapi_types.UUID,
) {
	orgID := organizationID.String()
	schedID := strings.TrimSpace(scheduleID.String())
	access, ok := h.resolveLinkedOrgAccess(w, r, orgID)
	if !ok {
		return
	}
	if schedID == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "schedule id is required")
		return
	}
	body, ok := readScheduleUpsertBody(w, r)
	if !ok {
		return
	}
	if h.liveData == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return
	}
	vaID := access.vaID
	orgCopy := orgID
	actorID := actorUserID(r)
	if err := h.tokens.Write(r.Context(), vaID, actorID, &orgCopy, actionUpdateSchedule, nil, func(token string) error {
		return h.liveData.UpdateSchedule(r.Context(), token, schedID, body)
	}); err != nil {
		writeScheduleWriteError(w, err, orgID)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ConnectionsHandler) UpdateVALiveScheduleFlightPlan(
	w http.ResponseWriter,
	r *http.Request,
	organizationID openapi_types.UUID,
	scheduleID openapi_types.UUID,
) {
	orgID := organizationID.String()
	schedID := strings.TrimSpace(scheduleID.String())
	access, ok := h.resolveLinkedOrgAccess(w, r, orgID)
	if !ok {
		return
	}
	if schedID == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "schedule id is required")
		return
	}
	body, ok := readScheduleFlightPlanBody(w, r)
	if !ok {
		return
	}
	if h.liveData == nil {
		response.WriteError(w, http.StatusServiceUnavailable, "LIVE_DISABLED", "Infinite Flight Live API is not configured")
		return
	}
	vaID := access.vaID
	orgCopy := orgID
	actorID := actorUserID(r)
	if err := h.tokens.Write(r.Context(), vaID, actorID, &orgCopy, actionUpdateScheduleFPL, nil, func(token string) error {
		return h.liveData.UpdateScheduleFlightPlan(r.Context(), token, schedID, body)
	}); err != nil {
		writeScheduleWriteError(w, err, orgID)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readScheduleUpsertBody(w http.ResponseWriter, r *http.Request) (json.RawMessage, bool) {
	payload, err := readJSONRequestBody(w, r)
	if !payload.ok {
		return nil, false
	}
	var parsed politburoapi.LiveScheduleUpsertBody
	if err := json.Unmarshal(payload.raw, &parsed); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "invalid JSON body")
		return nil, false
	}
	if isJSONObjectEmpty(payload.raw) {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "request body must include at least one field")
		return nil, false
	}
	encoded, err := encodeScheduleUpsertForUpstream(parsed)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
		return nil, false
	}
	return encoded, true
}

func readScheduleFlightPlanBody(w http.ResponseWriter, r *http.Request) (json.RawMessage, bool) {
	payload, err := readJSONRequestBody(w, r)
	if !payload.ok {
		return nil, false
	}
	var parsed politburoapi.LiveScheduleFlightPlanBody
	if err := json.Unmarshal(payload.raw, &parsed); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "invalid JSON body")
		return nil, false
	}
	if isJSONObjectEmpty(payload.raw) {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "request body must include at least one field")
		return nil, false
	}
	encoded, err := json.Marshal(parsed)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not encode request body")
		return nil, false
	}
	return encoded, true
}

type jsonBodyRead struct {
	raw json.RawMessage
	ok  bool
}

func readJSONRequestBody(w http.ResponseWriter, r *http.Request) (jsonBodyRead, error) {
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "could not read request body")
		return jsonBodyRead{}, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "request body is required")
		return jsonBodyRead{}, errors.New("empty body")
	}
	return jsonBodyRead{raw: raw, ok: true}, nil
}

func isJSONObjectEmpty(raw json.RawMessage) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	return len(obj) == 0
}

func actorUserID(r *http.Request) *string {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return nil
	}
	id := strings.TrimSpace(claims.PbUserID)
	if id == "" {
		return nil
	}
	return &id
}

func writeScheduleWriteError(w http.ResponseWriter, err error, organizationID string) {
	if errors.Is(err, valive.ErrOAuthRequired) {
		response.WriteError(w, http.StatusConflict, "IF_OAUTH_REQUIRED", "Connect Infinite Flight in Admin Live Mode first")
		return
	}
	writeLiveUpstreamError(w, err, "write infinite flight live schedule", organizationID)
}
