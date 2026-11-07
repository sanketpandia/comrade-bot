package live

import (
	"context"
	"net/http"
	"slices"
	"strings"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/transport/http/response"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type liveVADataClient interface {
	liveOrgClient
	ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error)
	ListOrganizationSchedules(context.Context, string, string, int, int, []string) (infiniteflight.LiveSchedulePage, error)
	ListAircraftFlights(context.Context, string, string, int, int) (infiniteflight.LiveAircraftFlightPage, error)
}

func (h *ConnectionsHandler) ListOrganizationAircraft(w http.ResponseWriter, r *http.Request, organizationID openapi_types.UUID) {
	orgID := organizationID.String()
	access, ok := h.resolveLinkedOrgAccess(w, r, orgID)
	if !ok {
		return
	}
	rows, err := h.liveData.ListOrganizationAircraft(r.Context(), access.accessToken, orgID)
	if err != nil {
		writeLiveUpstreamError(w, err, "list infinite flight organization aircraft", orgID)
		return
	}
	slices.SortFunc(rows, func(a, b infiniteflight.LiveAircraft) int {
		return a.FleetRank - b.FleetRank
	})
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapLiveAircraft(row, h.liveries))
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *ConnectionsHandler) ListOrganizationSchedules(
	w http.ResponseWriter,
	r *http.Request,
	organizationID openapi_types.UUID,
	params politburoapi.ListVALiveOrganizationSchedulesParams,
) {
	orgID := organizationID.String()
	access, ok := h.resolveLinkedOrgAccess(w, r, orgID)
	if !ok {
		return
	}
	page := intParam(params.Page, 1)
	pageSize := intParam(params.PageSize, 100)
	if pageSize > 100 {
		pageSize = 100
	}
	aircraftIDs := openapiUUIDsToStrings(params.AircraftId)
	result, err := h.liveData.ListOrganizationSchedules(r.Context(), access.accessToken, orgID, page, pageSize, aircraftIDs)
	if err != nil {
		writeLiveUpstreamError(w, err, "list infinite flight organization schedules", orgID)
		return
	}
	items := make([]map[string]any, 0, len(result.Items))
	for _, row := range result.Items {
		items = append(items, mapLiveSchedule(row))
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       items,
		"page":       result.Page,
		"pageSize":   result.PageSize,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
	})
}

func (h *ConnectionsHandler) ListAircraftFlights(
	w http.ResponseWriter,
	r *http.Request,
	organizationID openapi_types.UUID,
	aircraftID openapi_types.UUID,
	params politburoapi.ListVALiveAircraftFlightsParams,
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
		writeLiveUpstreamError(w, err, "list infinite flight organization aircraft for flight history", orgID)
		return
	}
	if !inFleet {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "aircraft not found in organisation fleet")
		return
	}
	page := intParam(params.Page, 1)
	pageSize := intParam(params.PageSize, 25)
	if pageSize > 100 {
		pageSize = 100
	}
	result, err := h.liveData.ListAircraftFlights(r.Context(), access.accessToken, acID, page, pageSize)
	if err != nil {
		writeLiveUpstreamError(w, err, "list infinite flight aircraft flights", orgID)
		return
	}
	items := make([]map[string]any, 0, len(result.Items))
	for _, row := range result.Items {
		items = append(items, mapLiveFlight(row))
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       items,
		"page":       result.Page,
		"pageSize":   result.PageSize,
		"totalCount": result.TotalCount,
		"totalPages": result.TotalPages,
	})
}

func (h *ConnectionsHandler) aircraftInOrganizationFleet(ctx context.Context, accessToken, orgID, aircraftID string) (bool, error) {
	rows, err := h.liveData.ListOrganizationAircraft(ctx, accessToken, orgID)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.ID == aircraftID {
			return true, nil
		}
	}
	return false, nil
}

func openapiUUIDsToStrings(ids *[]openapi_types.UUID) []string {
	if ids == nil || len(*ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(*ids))
	for _, id := range *ids {
		s := strings.TrimSpace(id.String())
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func mapLiveAircraft(row infiniteflight.LiveAircraft, lookup *gameliveries.Lookup) map[string]any {
	name := ""
	if lookup != nil {
		if resolved, outcome := lookup.Resolve(row.AircraftID, row.AircraftID); outcome != gameliveries.MatchMiss {
			name = resolved.AircraftName
		}
	}
	item := map[string]any{
		"id":                row.ID,
		"aircraftId":        row.AircraftID,
		"organizationId":    row.OrganizationID,
		"registration":      row.Registration,
		"status":            row.Status,
		"visibility":        row.Visibility,
		"fleetRank":         row.FleetRank,
		"isFleetActiveSlot": row.IsFleetActiveSlot,
		"createdAt":         row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"aircraftName":      name,
	}
	if row.FleetPriority != nil {
		item["fleetPriority"] = *row.FleetPriority
	}
	return item
}

func mapLiveSchedule(row infiniteflight.LiveSchedule) map[string]any {
	return map[string]any{
		"id":                    row.ID,
		"status":                row.Status,
		"callsign":              row.Callsign,
		"organizationId":        row.OrganizationID,
		"aircraftId":            row.AircraftID,
		"flightType":            row.FlightType,
		"originIcao":            row.OriginIcao,
		"destinationIcao":       row.DestinationIcao,
		"isTimeless":            row.IsTimeless,
		"scheduledDepartureUtc": row.ScheduledDepartureUtc.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"sequence":              row.Sequence,
	}
}

func mapLiveFlight(row infiniteflight.LiveAircraftFlight) map[string]any {
	crew := make([]map[string]any, 0, len(row.Crew))
	for _, member := range row.Crew {
		entry := map[string]any{"userId": member.UserID}
		if member.Username != nil {
			entry["username"] = *member.Username
		} else {
			entry["username"] = nil
		}
		crew = append(crew, entry)
	}
	item := map[string]any{
		"id":                  row.ID,
		"aircraftId":          row.AircraftID,
		"organizationId":      row.OrganizationID,
		"callsign":            row.Callsign,
		"status":              row.Status,
		"flightTimeMinutes":   row.FlightTimeMinutes,
		"airframeTimeMinutes": row.AirframeTimeMinutes,
		"crew":                crew,
	}
	if row.OriginIcao != nil {
		item["originIcao"] = *row.OriginIcao
	} else {
		item["originIcao"] = nil
	}
	if row.DestinationIcao != nil {
		item["destinationIcao"] = *row.DestinationIcao
	} else {
		item["destinationIcao"] = nil
	}
	if row.ActualDepartureUtc != nil {
		item["actualDepartureUtc"] = row.ActualDepartureUtc.UTC().Format("2006-01-02T15:04:05Z07:00")
	} else {
		item["actualDepartureUtc"] = nil
	}
	if row.ActualArrivalUtc != nil {
		item["actualArrivalUtc"] = row.ActualArrivalUtc.UTC().Format("2006-01-02T15:04:05Z07:00")
	} else {
		item["actualArrivalUtc"] = nil
	}
	return item
}
