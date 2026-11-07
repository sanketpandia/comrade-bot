package gameflights

import (
	"time"

	"github.com/google/uuid"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/transport/http/api/cachedresponse"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type activeFlightTrack struct {
	FlightID  string  `json:"flightId"`
	Callsign  string  `json:"callsign"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     int     `json:"speed"`
	Track     float64 `json:"track,omitempty"`
}

func tracksFromDomain(tracks []domainflights.Track) []activeFlightTrack {
	out := make([]activeFlightTrack, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, activeFlightTrack{
			FlightID:  track.FlightID,
			Callsign:  track.Callsign,
			Latitude:  track.Latitude,
			Longitude: track.Longitude,
			Speed:     track.Speed,
			Track:     track.Track,
		})
	}
	return out
}

func activeFlightsResponse(result domainflights.ListActiveTracksResult) cachedresponse.Response[activeFlightTrack] {
	return cachedresponse.Response[activeFlightTrack]{
		Data: cachedresponse.Data[activeFlightTrack]{
			AvailableFilters: cachedFiltersFromDomain(result.AvailableFilters),
			Result:           tracksFromDomain(result.Tracks),
			Meta: cachedresponse.Meta{
				LastCached:          result.LastCached,
				RefreshIntervalMins: int(cache.FlightsRefreshInterval / time.Minute),
			},
			Pagination: &cachedresponse.Pagination{
				TotalLength: result.TotalLength,
				PageLength:  result.PageLength,
				PageNumber:  result.PageNumber,
			},
		},
	}
}

func cachedFiltersFromDomain(filters []domainflights.AvailableFilter) []cachedresponse.Filter {
	out := make([]cachedresponse.Filter, 0, len(filters))
	for _, filter := range filters {
		out = append(out, cachedresponse.Filter{
			Name:    filter.Name,
			Type:    filter.Type,
			Desc:    filter.Desc,
			Current: filter.Current,
			Default: filter.Default,
			Options: filter.Options,
		})
	}
	return out
}

func activeFlightRecordFromDomain(record domainflights.Record, flightID openapi_types.UUID, lookup *gameliveries.Lookup) politburoapi.ActiveFlightRecord {
	raw := record.Raw
	norm := domainflights.NormalizeRecord(record, lookup)
	lastReport := raw.LastReport
	if !raw.LastReportValid {
		lastReport = raw.ObservedAt
	}

	apiRaw := politburoapi.ActiveFlightRaw{
		FlightId:       flightID,
		Callsign:       raw.Callsign,
		Latitude:       domainflights.RoundedLatitude(raw.Latitude),
		Longitude:      domainflights.RoundedLongitude(raw.Longitude),
		Altitude:       domainflights.RoundedAltitude(raw.Altitude),
		Speed:          domainflights.RoundedSpeed(raw.Speed),
		VerticalSpeed:  domainflights.RoundedVerticalSpeed(raw.VerticalSpeed),
		Track:          domainflights.RoundedTrack(raw.Track),
		LastReport:     lastReport,
		PilotState:     raw.PilotState,
		IsConnected:    raw.IsConnected,
		SessionId:      raw.SessionID,
		NormalizedName: raw.NormalizedName,
		Username:       raw.Username,
		VirtualOrganization: raw.VirtualOrganization,
	}
	if raw.UserID != "" {
		apiRaw.UserId = &raw.UserID
	}
	if raw.AircraftID != "" {
		apiRaw.AircraftId = &raw.AircraftID
	}
	if raw.LiveryID != "" {
		apiRaw.LiveryId = &raw.LiveryID
	}

	apiNormalized := politburoapi.ActiveFlightNormalized{
		Speed:         norm.Speed,
		VerticalSpeed: norm.VerticalSpeed,
		PilotState:    politburoapi.PilotStateName(norm.PilotState),
		IsConnected:   politburoapi.ActiveFlightNormalizedIsConnected(norm.IsConnected),
	}
	if norm.NormalizedCallsign != "" {
		apiNormalized.NormalizedCallsign = &norm.NormalizedCallsign
	}
	if norm.AircraftName != "" {
		apiNormalized.AircraftName = &norm.AircraftName
	}
	if norm.LiveryName != "" {
		apiNormalized.LiveryName = &norm.LiveryName
	}
	if norm.Departure != "" {
		apiNormalized.Departure = &norm.Departure
	}
	if norm.Destination != "" {
		apiNormalized.Destination = &norm.Destination
	}

	apiActions := politburoapi.ActiveFlightActions{
		FplSync: politburoapi.ActiveFlightFPLSync{
			Required: record.Actions.FPLSync.Required,
		},
	}
	if !record.Actions.FPLSync.LastAt.IsZero() {
		lastAt := record.Actions.FPLSync.LastAt
		apiActions.FplSync.LastAt = &lastAt
	}

	apiLinks := politburoapi.ActiveFlightLinks{}
	if len(record.Links.VirtualAirlines) > 0 {
		apiLinks.VirtualAirlines = matchedVirtualAirlinesFromDomain(record.Links.VirtualAirlines)
	}
	if len(record.Links.Users) > 0 {
		apiLinks.Users = &record.Links.Users
	}

	apiAnalytics := politburoapi.ActiveFlightAnalytics{}
	if record.Analytics.FlightState != nil {
		apiAnalytics.FlightState = activeFlightStateFromDomain(*record.Analytics.FlightState)
	}

	return politburoapi.ActiveFlightRecord{
		Raw:        apiRaw,
		Normalized: apiNormalized,
		Analytics:  apiAnalytics,
		Actions:    apiActions,
		Links:      apiLinks,
		History:    politburoapi.ActiveFlightHistory{},
	}
}

func activeFlightStateFromDomain(state flightstate.Summary) *politburoapi.ActiveFlightState {
	out := &politburoapi.ActiveFlightState{
		Phase:       politburoapi.ActiveFlightStatePhase(state.Phase),
		Activity:    politburoapi.ActiveFlightStateActivity(state.Activity),
		DataQuality: politburoapi.ActiveFlightStateDataQuality(state.DataQuality),
		Confidence:  state.Confidence,
		PhaseSince:  state.PhaseSince,
	}
	out.ApPlus.Candidate = state.APPlus.Candidate
	out.ApPlus.Score = state.APPlus.Score
	out.ApPlus.Since = state.APPlus.Since
	if len(state.APPlus.Reasons) > 0 {
		out.ApPlus.Reasons = &state.APPlus.Reasons
	}
	if state.PreviousPhase != "" {
		p := string(state.PreviousPhase)
		out.PreviousPhase = &p
	}
	if len(state.Reasons) > 0 {
		out.Reasons = &state.Reasons
	}
	if state.Features != nil {
		out.Features = flightStateFeaturesFromSnapshot(*state.Features)
	}
	return out
}

func enrichActiveFlightStateFeatures(record *politburoapi.ActiveFlightRecord, history flightstate.History) {
	if record == nil || record.Analytics.FlightState == nil {
		return
	}
	features := history.State.Features
	if features.ElapsedSec == 0 && features.DistanceNM == 0 && features.GapClass == "" {
		return
	}
	record.Analytics.FlightState.Features = flightStateFeaturesFromSnapshot(features)
}

func flightStateFeaturesFromSnapshot(features flightstate.WindowFeatureSnapshot) *struct {
	AltDeltaFt            *float64                                        `json:"altDeltaFt,omitempty"`
	AltStable             *bool                                           `json:"altStable,omitempty"`
	AvgVerticalFpm        *float64                                        `json:"avgVerticalFpm,omitempty"`
	DistanceNm            *float64                                        `json:"distanceNm,omitempty"`
	ElapsedSec            *float64                                        `json:"elapsedSec,omitempty"`
	Fresh                 *bool                                           `json:"fresh,omitempty"`
	GapClass              *politburoapi.ActiveFlightStateFeaturesGapClass `json:"gapClass,omitempty"`
	LastReportAdvancing   *bool                                           `json:"lastReportAdvancing,omitempty"`
	Moved                 *bool                                           `json:"moved,omitempty"`
	ProgressRatio         *float64                                        `json:"progressRatio,omitempty"`
	SpeedDeltaKt          *float64                                        `json:"speedDeltaKt,omitempty"`
	StationaryDurationSec *float64                                        `json:"stationaryDurationSec,omitempty"`
	StationarySamples     *int                                            `json:"stationarySamples,omitempty"`
	SustainedClimbCount   *int                                            `json:"sustainedClimbCount,omitempty"`
	SustainedDescentCount *int                                            `json:"sustainedDescentCount,omitempty"`
	TrackDeltaDeg         *float64                                        `json:"trackDeltaDeg,omitempty"`
} {
	return &struct {
		AltDeltaFt            *float64                                        `json:"altDeltaFt,omitempty"`
		AltStable             *bool                                           `json:"altStable,omitempty"`
		AvgVerticalFpm        *float64                                        `json:"avgVerticalFpm,omitempty"`
		DistanceNm            *float64                                        `json:"distanceNm,omitempty"`
		ElapsedSec            *float64                                        `json:"elapsedSec,omitempty"`
		Fresh                 *bool                                           `json:"fresh,omitempty"`
		GapClass              *politburoapi.ActiveFlightStateFeaturesGapClass `json:"gapClass,omitempty"`
		LastReportAdvancing   *bool                                           `json:"lastReportAdvancing,omitempty"`
		Moved                 *bool                                           `json:"moved,omitempty"`
		ProgressRatio         *float64                                        `json:"progressRatio,omitempty"`
		SpeedDeltaKt          *float64                                        `json:"speedDeltaKt,omitempty"`
		StationaryDurationSec *float64                                        `json:"stationaryDurationSec,omitempty"`
		StationarySamples     *int                                            `json:"stationarySamples,omitempty"`
		SustainedClimbCount   *int                                            `json:"sustainedClimbCount,omitempty"`
		SustainedDescentCount *int                                            `json:"sustainedDescentCount,omitempty"`
		TrackDeltaDeg         *float64                                        `json:"trackDeltaDeg,omitempty"`
	}{
		ElapsedSec:            &features.ElapsedSec,
		DistanceNm:            &features.DistanceNM,
		AltDeltaFt:            &features.AltDeltaFt,
		AvgVerticalFpm:        &features.AvgVerticalFpm,
		SpeedDeltaKt:          &features.SpeedDeltaKt,
		TrackDeltaDeg:         &features.TrackDeltaDeg,
		Moved:                 &features.Moved,
		AltStable:             &features.AltStable,
		ProgressRatio:         &features.ProgressRatio,
		GapClass:              gapClassPtr(features.GapClass),
		Fresh:                 &features.Fresh,
		StationarySamples:     &features.StationarySamples,
		StationaryDurationSec: &features.StationaryDurationSec,
		SustainedClimbCount:   &features.SustainedClimbCount,
		SustainedDescentCount: &features.SustainedDescentCount,
		LastReportAdvancing:   &features.LastReportAdvancing,
	}
}

func gapClassPtr(c flightstate.GapClass) *politburoapi.ActiveFlightStateFeaturesGapClass {
	if c == "" {
		return nil
	}
	v := politburoapi.ActiveFlightStateFeaturesGapClass(c)
	return &v
}

func matchedVirtualAirlinesFromDomain(matches []virtualairlines.VirtualAirlineMatch) *[]politburoapi.ActiveFlightVirtualAirlineMatch {
	out := make([]politburoapi.ActiveFlightVirtualAirlineMatch, 0, len(matches))
	for _, match := range matches {
		parsed, err := uuid.Parse(match.VAID)
		if err != nil {
			continue
		}
		out = append(out, politburoapi.ActiveFlightVirtualAirlineMatch{
			VaId:   openapi_types.UUID(parsed),
			VaCode: match.VACode,
		})
	}
	return &out
}

func activeFlightPlanFromDomain(plan domainflights.CachedFlightPlan) politburoapi.ActiveFlightPlanData {
	out := politburoapi.ActiveFlightPlanData{
		FetchedAt: plan.FetchedAt,
		Absent:    &plan.Absent,
	}
	if plan.FlightPlanID != "" {
		if id, err := uuid.Parse(plan.FlightPlanID); err == nil {
			parsed := openapi_types.UUID(id)
			out.FlightPlanId = &parsed
		}
	}
	if plan.FlightPlanType != nil {
		out.FlightPlanType = plan.FlightPlanType
	}
	if len(plan.Waypoints) > 0 {
		waypoints := make([]politburoapi.ActiveFlightPlanWaypoint, 0, len(plan.Waypoints))
		for _, wp := range plan.Waypoints {
			waypoints = append(waypoints, politburoapi.ActiveFlightPlanWaypoint{
				Name:      wp.Name,
				Latitude:  wp.Latitude,
				Longitude: wp.Longitude,
				Altitude:  wp.Altitude,
			})
		}
		out.Waypoints = &waypoints
	}
	return out
}

func activeFlightPositionHistoryFromDomain(trail domainflights.PositionTrail) politburoapi.ActiveFlightPositionHistoryData {
	fixes := make([]politburoapi.ActiveFlightPositionFix, 0, len(trail.Fixes))
	for _, fix := range trail.Fixes {
		fixes = append(fixes, politburoapi.ActiveFlightPositionFix{
			At:        fix.At,
			Latitude:  fix.Latitude,
			Longitude: fix.Longitude,
			Altitude:  fix.Altitude,
			Speed:     fix.Speed,
		})
	}
	return politburoapi.ActiveFlightPositionHistoryData{Fixes: fixes}
}
