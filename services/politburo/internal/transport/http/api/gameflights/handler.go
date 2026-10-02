package gameflights

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/cache"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/transport/http/api/cachedresponse"
	"infinite-experiment/politburo/internal/transport/http/response"
)

type Handler struct {
	reader  *domainflights.Reader
	metrics *metrics.Registry
}

type activeFlightTrack struct {
	FlightID  string  `json:"flightId"`
	Callsign  string  `json:"callsign"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     int     `json:"speed"`
	Track     float64 `json:"track,omitempty"`
}

func NewHandler(reader *domainflights.Reader, metricsRegistry *metrics.Registry) *Handler {
	return &Handler{reader: reader, metrics: metricsRegistry}
}

func (h *Handler) GetActiveFlights(w http.ResponseWriter, r *http.Request, normalizedServerName string, params politburoapi.GetActiveFlightsParams) {
	if normalizedServerName == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "normalizedServerName is required")
		return
	}

	pageNumber, pageLength, err := domainflights.ParsePaginationQuery(r.URL.Query())
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
		return
	}

	filterQuery, err := filterQueryFromParams(params)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", "invalid pilotState value")
		return
	}

	result, err := h.reader.ListActiveTracks(r.Context(), domainflights.ListActiveTracksInput{
		Server:     normalizedServerName,
		PageNumber: pageNumber,
		PageLength: pageLength,
		Filters:    filterQuery,
	})
	if err != nil {
		writeListActiveTracksError(w, r, normalizedServerName, filterQuery, err)
		return
	}

	if filterQueryHasValues(filterQuery) {
		h.metrics.FlightsFilteredTotal.WithLabelValues(normalizedServerName, "active").Add(float64(len(result.Tracks)))
		slog.Info("active flights filter",
			"endpoint", "active",
			"serverId", normalizedServerName,
			"pilotState", filterQuery.PilotStates,
			"userName", filterQuery.UserName,
			"callSign", filterQuery.CallSign,
			"count", len(result.Tracks),
		)
	}

	tracks := make([]activeFlightTrack, 0, len(result.Tracks))
	for _, track := range result.Tracks {
		tracks = append(tracks, activeFlightTrack{
			FlightID:  track.FlightID,
			Callsign:  track.Callsign,
			Latitude:  track.Latitude,
			Longitude: track.Longitude,
			Speed:     track.Speed,
			Track:     track.Track,
		})
	}

	response.WriteJSON(w, http.StatusOK, cachedresponse.Response[activeFlightTrack]{
		Data: cachedresponse.Data[activeFlightTrack]{
			AvailableFilters: toCachedFilters(result.AvailableFilters),
			Result:           tracks,
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
	})
}

func filterQueryFromParams(params politburoapi.GetActiveFlightsParams) (domainflights.FilterQuery, error) {
	var pilotNames []string
	if params.PilotState != nil {
		pilotNames = make([]string, 0, len(*params.PilotState))
		for _, state := range *params.PilotState {
			pilotNames = append(pilotNames, string(state))
		}
	}
	states, err := domainflights.ParseFilterPilotStates(pilotNames)
	if err != nil {
		return domainflights.FilterQuery{}, err
	}
	userName := ""
	if params.UserName != nil {
		userName = *params.UserName
	}
	callSign := ""
	if params.CallSign != nil {
		callSign = *params.CallSign
	}
	userName, callSign = domainflights.NormalizeFilterStrings(userName, callSign)
	return domainflights.FilterQuery{
		PilotStates: states,
		UserName:    userName,
		CallSign:    callSign,
	}, nil
}

func filterQueryHasValues(query domainflights.FilterQuery) bool {
	return len(query.PilotStates) > 0 || query.UserName != "" || query.CallSign != ""
}

func writeListActiveTracksError(w http.ResponseWriter, r *http.Request, server string, query domainflights.FilterQuery, err error) {
	switch {
	case errors.Is(err, domainflights.ErrUnknownServer), errors.Is(err, domainflights.ErrInvalidFilter):
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
	case errors.Is(err, domainflights.ErrCacheMiss):
		slog.Warn("active flights cache miss", "serverId", server, "error", err)
		response.WriteError(w, http.StatusServiceUnavailable, "ACTIVE_FLIGHTS_CACHE_UNAVAILABLE", "active flights cache is unavailable")
	case errors.Is(err, domainflights.ErrCacheCorrupt), errors.Is(err, domainflights.ErrCacheRead), errors.Is(err, domainflights.ErrSessionNames):
		if errors.Is(err, domainflights.ErrCacheCorrupt) {
			slog.Error("read active flights cache", "error", "lastCached is missing", "serverId", server)
		} else {
			slog.Error("read active flights cache", "error", err, "serverId", server)
		}
		response.WriteError(w, http.StatusInternalServerError, "ACTIVE_FLIGHTS_CACHE_UNAVAILABLE", "active flights cache is unavailable")
	default:
		slog.Error("list active flights", "error", err, "serverId", server, "filters", query)
		response.WriteError(w, http.StatusInternalServerError, "ACTIVE_FLIGHTS_CACHE_UNAVAILABLE", "active flights cache is unavailable")
	}
}

func toCachedFilters(filters []domainflights.AvailableFilter) []cachedresponse.Filter {
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
