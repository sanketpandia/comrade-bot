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

func activeFlightRecordFromDomain(flight domainflights.Flight, flightID openapi_types.UUID) politburoapi.ActiveFlightRecord {
	record := politburoapi.ActiveFlightRecord{
		FlightId:           flightID,
		Callsign:           flight.Callsign,
		NormalizedCallsign: optionalString(flight.NormalizedCallsign),
		Latitude:           flight.Latitude,
		Longitude:          flight.Longitude,
		Altitude:           flight.Altitude,
		Speed:              flight.Speed,
		VerticalSpeed:      flight.VerticalSpeed,
		Track:              flight.Track,
		LastReport:         flight.LastReport,
		PilotState:         flight.PilotState,
		IsConnected:        flight.IsConnected,
		SessionId:          flight.SessionID,
		NormalizedName:     flight.NormalizedName,
		Normalized: politburoapi.ActiveFlightNormalized{
			Speed:         flight.Normalized.Speed,
			VerticalSpeed: flight.Normalized.VerticalSpeed,
			PilotState:    politburoapi.PilotStateName(flight.Normalized.PilotState),
			IsConnected:   politburoapi.ActiveFlightNormalizedIsConnected(flight.Normalized.IsConnected),
		},
		Username:            flight.Username,
		VirtualOrganization: flight.VirtualOrganization,
	}
	if flight.UserID != "" {
		record.UserId = &flight.UserID
	}
	if flight.AircraftID != "" {
		record.AircraftId = &flight.AircraftID
	}
	if flight.LiveryID != "" {
		record.LiveryId = &flight.LiveryID
	}
	if flight.AircraftName != "" {
		record.AircraftName = &flight.AircraftName
	}
	if flight.LiveryName != "" {
		record.LiveryName = &flight.LiveryName
	}
	if flight.PathSync != nil {
		pathSync := politburoapi.ActiveFlightPathSync{
			FplSyncRequired: flight.PathSync.FPLSyncRequired,
		}
		if !flight.PathSync.LastFPLSyncAt.IsZero() {
			pathSync.LastFPLSyncAt = &flight.PathSync.LastFPLSyncAt
		}
		record.PathSync = &pathSync
	}
	if len(flight.MatchedVirtualAirlines) > 0 {
		record.MatchedVirtualAirlines = matchedVirtualAirlinesFromDomain(flight.MatchedVirtualAirlines)
	}
	return record
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
