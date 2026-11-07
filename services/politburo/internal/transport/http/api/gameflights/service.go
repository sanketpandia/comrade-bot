package gameflights

import (
	"context"
	"log/slog"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/transport/http/api/cachedresponse"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Service loads active flight data from cache and shapes API responses.
type Service struct {
	reader      *domainflights.Reader
	planFetcher *domainflights.PlanFetcher
	lookup      *gameliveries.Lookup
	metrics     *metrics.Registry
}

func NewService(reader *domainflights.Reader, planFetcher *domainflights.PlanFetcher, lookup *gameliveries.Lookup, metricsRegistry *metrics.Registry) *Service {
	return &Service{reader: reader, planFetcher: planFetcher, lookup: lookup, metrics: metricsRegistry}
}

func (s *Service) GetActiveFlight(ctx context.Context, flightID openapi_types.UUID) (politburoapi.ActiveFlightRecordResponse, error) {
	flight, err := s.reader.GetActiveFlight(ctx, flightID.String())
	if err != nil {
		return politburoapi.ActiveFlightRecordResponse{}, err
	}
	record := activeFlightRecordFromDomain(flight, flightID, s.lookup)
	enrichActiveFlightStateFeatures(&record, flight.History)
	return politburoapi.ActiveFlightRecordResponse{Data: record}, nil
}

func (s *Service) GetActiveFlightPlan(ctx context.Context, flightID openapi_types.UUID) (politburoapi.ActiveFlightPlanResponse, error) {
	if s.planFetcher == nil {
		return politburoapi.ActiveFlightPlanResponse{}, domainflights.ErrCacheRead
	}
	flight, err := s.reader.GetActiveFlight(ctx, flightID.String())
	if err != nil {
		return politburoapi.ActiveFlightPlanResponse{}, err
	}
	plan, err := s.planFetcher.GetOrFetch(ctx, flight.Raw.SessionID, flightID.String())
	if err != nil {
		return politburoapi.ActiveFlightPlanResponse{}, err
	}
	if plan.Absent {
		return politburoapi.ActiveFlightPlanResponse{}, domainflights.ErrCacheMiss
	}
	return politburoapi.ActiveFlightPlanResponse{Data: activeFlightPlanFromDomain(plan)}, nil
}

func (s *Service) GetActiveFlightPositionHistory(ctx context.Context, flightID openapi_types.UUID) (politburoapi.ActiveFlightPositionHistoryResponse, error) {
	trail, err := s.reader.GetPositionTrail(ctx, flightID.String())
	if err != nil {
		return politburoapi.ActiveFlightPositionHistoryResponse{}, err
	}
	return politburoapi.ActiveFlightPositionHistoryResponse{Data: activeFlightPositionHistoryFromDomain(trail)}, nil
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
