package gameflights

import (
	"errors"
	"log/slog"
	"net/http"

	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/transport/http/response"
)

func writeGetActiveFlightError(w http.ResponseWriter, flightID string, err error) {
	switch {
	case errors.Is(err, domainflights.ErrInvalidFilter):
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
	case errors.Is(err, domainflights.ErrCacheMiss):
		slog.Warn("active flight cache miss", "flightId", flightID)
		response.WriteError(w, http.StatusNotFound, "ACTIVE_FLIGHT_NOT_FOUND", "active flight not found")
	case errors.Is(err, domainflights.ErrCacheRead):
		slog.Error("read active flight cache", "error", err, "flightId", flightID)
		response.WriteError(w, http.StatusInternalServerError, "ACTIVE_FLIGHT_CACHE_UNAVAILABLE", "active flight cache is unavailable")
	default:
		slog.Error("get active flight", "error", err, "flightId", flightID)
		response.WriteError(w, http.StatusInternalServerError, "ACTIVE_FLIGHT_CACHE_UNAVAILABLE", "active flight cache is unavailable")
	}
}

func writeListActiveTracksError(w http.ResponseWriter, server string, query domainflights.FilterQuery, err error) {
	switch {
	case errors.Is(err, domainflights.ErrUnknownServer), errors.Is(err, domainflights.ErrInvalidFilter), errors.Is(err, domainflights.ErrUnknownVACode):
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

func writeListActiveTracksRequestError(w http.ResponseWriter, err error) {
	if isInvalidListRequest(err) {
		msg := err.Error()
		switch {
		case errors.Is(err, domainflights.ErrInvalidFilter):
			msg = "invalid pilotState value"
		case errors.Is(err, domainflights.ErrUnknownVACode):
			msg = "unknown virtual airline code"
		}
		response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", msg)
		return
	}
	response.WriteError(w, http.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
}
