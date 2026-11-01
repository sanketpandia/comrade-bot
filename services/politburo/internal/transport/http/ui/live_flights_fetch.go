package ui

import (
	"context"
	"fmt"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type liveFlightsUpstreamData struct {
	Aircraft  []infiniteflight.LiveAircraft
	Flights   []infiniteflight.LiveAircraftFlight
	Truncated bool
}

type liveFlightsUpstreamClient interface {
	ListAircraftFlights(context.Context, string, string, int, int) (infiniteflight.LiveAircraftFlightPage, error)
}

type liveFlightsFetchClient interface {
	liveSchedulesUpstreamClient
	liveFlightsUpstreamClient
}

func fetchLiveFlightsUpstream(
	ctx context.Context,
	live liveFlightsFetchClient,
	accessToken, organizationID, filterAircraftID string,
) (liveFlightsUpstreamData, error) {
	aircraft, err := live.ListOrganizationAircraft(ctx, accessToken, organizationID)
	if err != nil {
		return liveFlightsUpstreamData{}, fmt.Errorf("list organization aircraft: %w", err)
	}
	data := liveFlightsUpstreamData{Aircraft: aircraft}
	if filterAircraftID == "" {
		return data, nil
	}
	flights, truncated, err := fetchAircraftFlightsPages(ctx, live, accessToken, filterAircraftID)
	if err != nil {
		return liveFlightsUpstreamData{}, fmt.Errorf("list aircraft flights: %w", err)
	}
	data.Flights = flights
	data.Truncated = truncated
	return data, nil
}

func fetchAircraftFlightsPages(
	ctx context.Context,
	live liveFlightsUpstreamClient,
	accessToken, aircraftID string,
) (items []infiniteflight.LiveAircraftFlight, truncated bool, err error) {
	for page := 1; page <= liveFlightsMaxPages; page++ {
		result, err := live.ListAircraftFlights(ctx, accessToken, aircraftID, page, liveFlightsPageSize)
		if err != nil {
			return nil, false, err
		}
		items = append(items, result.Items...)
		if page >= result.TotalPages {
			return items, false, nil
		}
		if page == liveFlightsMaxPages && result.TotalPages > liveFlightsMaxPages {
			return items, true, nil
		}
	}
	return items, false, nil
}
