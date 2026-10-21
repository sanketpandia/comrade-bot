package gameflights

import (
	"context"
	"log/slog"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/transport/http/api/cachedresponse"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Service loads active flight data from cache and shapes API responses.
type Service struct {
	reader  *domainflights.Reader
	metrics *metrics.Registry
}

func NewService(reader *domainflights.Reader, metricsRegistry *metrics.Registry) *Service {
	return &Service{reader: reader, metrics: metricsRegistry}
}

func (s *Service) GetActiveFlight(ctx context.Context, flightID openapi_types.UUID) (politburoapi.ActiveFlightRecordResponse, error) {
	flight, err := s.reader.GetActiveFlight(ctx, flightID.String())
	if err != nil {
		return politburoapi.ActiveFlightRecordResponse{}, err
	}
	history, err := s.reader.GetFlightHistory(ctx, flightID.String())
	if err != nil {
		return politburoapi.ActiveFlightRecordResponse{}, err
	}
	record := activeFlightRecordFromDomain(flight, flightID)
	enrichActiveFlightStateFeatures(&record, history)
	return politburoapi.ActiveFlightRecordResponse{Data: record}, nil
}

func (s *Service) ListActiveTracks(ctx context.Context, input domainflights.ListActiveTracksInput) (cachedresponse.Response[activeFlightTrack], error) {
	result, err := s.reader.ListActiveTracks(ctx, input)
	if err != nil {
		return cachedresponse.Response[activeFlightTrack]{}, err
	}
	if input.Filters.HasValues() {
		s.metrics.FlightsFilteredTotal.WithLabelValues(input.Server, "active").Add(float64(len(result.Tracks)))
		slog.Info("active flights filter",
			"endpoint", "active",
			"serverId", input.Server,
			"pilotState", input.Filters.PilotStates,
			"userName", input.Filters.UserName,
			"callSign", input.Filters.CallSign,
			"vaCode", input.Filters.VACode,
			"count", len(result.Tracks),
		)
	}
	return activeFlightsResponse(result), nil
}
