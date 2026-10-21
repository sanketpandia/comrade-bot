package gameflights

import (
	"time"

	"github.com/google/uuid"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
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

func activeFlightRecordFromDomain(record domainflights.Record, flightID openapi_types.UUID) politburoapi.ActiveFlightRecord {
	raw := record.Raw
	derived := record.Derived
	lastReport := raw.LastReport
	if !raw.LastReportValid {
		lastReport = raw.ObservedAt
	}

	apiRecord := politburoapi.ActiveFlightRecord{
		FlightId:           flightID,
		Callsign:           raw.Callsign,
		NormalizedCallsign: optionalString(derived.NormalizedCallsign),
		Latitude:           domainflights.RoundedLatitude(raw.Latitude),
		Longitude:          domainflights.RoundedLongitude(raw.Longitude),
		Altitude:           domainflights.RoundedAltitude(raw.Altitude),
		Speed:              domainflights.RoundedSpeed(raw.Speed),
		VerticalSpeed:      domainflights.RoundedVerticalSpeed(raw.VerticalSpeed),
		Track:              domainflights.RoundedTrack(raw.Track),
		LastReport:         lastReport,
		PilotState:         raw.PilotState,
		IsConnected:        raw.IsConnected,
		SessionId:          raw.SessionID,
		NormalizedName:     raw.NormalizedName,
		Normalized: politburoapi.ActiveFlightNormalized{
			Speed:         derived.Normalized.Speed,
			VerticalSpeed: derived.Normalized.VerticalSpeed,
			PilotState:    politburoapi.PilotStateName(derived.Normalized.PilotState),
			IsConnected:   politburoapi.ActiveFlightNormalizedIsConnected(derived.Normalized.IsConnected),
		},
		Username:            raw.Username,
		VirtualOrganization: raw.VirtualOrganization,
	}
	if raw.UserID != "" {
		apiRecord.UserId = &raw.UserID
	}
	if raw.AircraftID != "" {
		apiRecord.AircraftId = &raw.AircraftID
	}
	if raw.LiveryID != "" {
		apiRecord.LiveryId = &raw.LiveryID
	}
	if derived.AircraftName != "" {
		apiRecord.AircraftName = &derived.AircraftName
	}
	if derived.LiveryName != "" {
		apiRecord.LiveryName = &derived.LiveryName
	}
	pathSync := derived.PathSync
	apiPathSync := politburoapi.ActiveFlightPathSync{
		FplSyncRequired: pathSync.FPLSyncRequired,
	}
	if !pathSync.LastFPLSyncAt.IsZero() {
		apiPathSync.LastFPLSyncAt = &pathSync.LastFPLSyncAt
	}
	apiRecord.PathSync = &apiPathSync
	if len(derived.MatchedVirtualAirlines) > 0 {
		apiRecord.MatchedVirtualAirlines = matchedVirtualAirlinesFromDomain(derived.MatchedVirtualAirlines)
	}
	if derived.FlightState != nil {
		apiRecord.FlightState = activeFlightStateFromDomain(*derived.FlightState)
	}
	if len(derived.Users) > 0 {
		apiRecord.Users = &derived.Users
	}
	return apiRecord
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
	if record == nil || record.FlightState == nil {
		return
	}
	features := history.State.Features
	if features.ElapsedSec == 0 && features.DistanceNM == 0 && features.GapClass == "" {
		return
	}
	record.FlightState.Features = flightStateFeaturesFromSnapshot(features)
}

func flightStateFeaturesFromSnapshot(features flightstate.WindowFeatureSnapshot) *struct {
	AltDeltaFt            *float64                           `json:"altDeltaFt,omitempty"`
	AltStable             *bool                              `json:"altStable,omitempty"`
	AvgVerticalFpm        *float64                           `json:"avgVerticalFpm,omitempty"`
	DistanceNm            *float64                           `json:"distanceNm,omitempty"`
	ElapsedSec            *float64                           `json:"elapsedSec,omitempty"`
	Fresh                 *bool                              `json:"fresh,omitempty"`
	GapClass              *politburoapi.ActiveFlightStateFeaturesGapClass `json:"gapClass,omitempty"`
	LastReportAdvancing   *bool                              `json:"lastReportAdvancing,omitempty"`
	Moved                 *bool                              `json:"moved,omitempty"`
	ProgressRatio         *float64                           `json:"progressRatio,omitempty"`
	SpeedDeltaKt          *float64                           `json:"speedDeltaKt,omitempty"`
	StationaryDurationSec *float64                           `json:"stationaryDurationSec,omitempty"`
	StationarySamples     *int                               `json:"stationarySamples,omitempty"`
	SustainedClimbCount   *int                               `json:"sustainedClimbCount,omitempty"`
	SustainedDescentCount *int                               `json:"sustainedDescentCount,omitempty"`
	TrackDeltaDeg         *float64                           `json:"trackDeltaDeg,omitempty"`
} {
	return &struct {
		AltDeltaFt            *float64                           `json:"altDeltaFt,omitempty"`
		AltStable             *bool                              `json:"altStable,omitempty"`
		AvgVerticalFpm        *float64                           `json:"avgVerticalFpm,omitempty"`
		DistanceNm            *float64                           `json:"distanceNm,omitempty"`
		ElapsedSec            *float64                           `json:"elapsedSec,omitempty"`
		Fresh                 *bool                              `json:"fresh,omitempty"`
		GapClass              *politburoapi.ActiveFlightStateFeaturesGapClass `json:"gapClass,omitempty"`
		LastReportAdvancing   *bool                              `json:"lastReportAdvancing,omitempty"`
		Moved                 *bool                              `json:"moved,omitempty"`
		ProgressRatio         *float64                           `json:"progressRatio,omitempty"`
		SpeedDeltaKt          *float64                           `json:"speedDeltaKt,omitempty"`
		StationaryDurationSec *float64                           `json:"stationaryDurationSec,omitempty"`
		StationarySamples     *int                               `json:"stationarySamples,omitempty"`
		SustainedClimbCount   *int                               `json:"sustainedClimbCount,omitempty"`
		SustainedDescentCount *int                               `json:"sustainedDescentCount,omitempty"`
		TrackDeltaDeg         *float64                           `json:"trackDeltaDeg,omitempty"`
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

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
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
