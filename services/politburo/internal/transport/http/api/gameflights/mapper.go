package gameflights

import (
	"time"

	"github.com/google/uuid"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
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
		lastReport = time.Time{}
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
	if pathSync.FPLSyncRequired || !pathSync.LastFPLSyncAt.IsZero() {
		apiPathSync := politburoapi.ActiveFlightPathSync{
			FplSyncRequired: pathSync.FPLSyncRequired,
		}
		if !pathSync.LastFPLSyncAt.IsZero() {
			apiPathSync.LastFPLSyncAt = &pathSync.LastFPLSyncAt
		}
		apiRecord.PathSync = &apiPathSync
	}
	if len(derived.MatchedVirtualAirlines) > 0 {
		apiRecord.MatchedVirtualAirlines = matchedVirtualAirlinesFromDomain(derived.MatchedVirtualAirlines)
	}
	return apiRecord
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
