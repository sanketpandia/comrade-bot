package ui

import (
	"context"
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func TestFetchLiveFlightsUpstreamSkipsFlightsWithoutAircraftFilter(t *testing.T) {
	stub := &flightsLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1"}},
		flights: []infiniteflight.LiveAircraftFlight{{
			ID: "flight-1",
		}},
	}
	data, err := fetchLiveFlightsUpstream(context.Background(), stub, "tok", "org-1", "")
	if err != nil {
		t.Fatalf("fetchLiveFlightsUpstream() error = %v", err)
	}
	if stub.flightCalls != 0 {
		t.Fatalf("flightCalls = %d, want 0", stub.flightCalls)
	}
	if len(data.Flights) != 0 {
		t.Fatalf("flights = %+v", data.Flights)
	}
}

func TestFetchLiveFlightsUpstreamLoadsFlightsWhenFiltered(t *testing.T) {
	stub := &flightsLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1"}},
		flights: []infiniteflight.LiveAircraftFlight{{
			ID:         "flight-1",
			AircraftID: "ac-1",
		}},
	}
	data, err := fetchLiveFlightsUpstream(context.Background(), stub, "tok", "org-1", "ac-1")
	if err != nil {
		t.Fatalf("fetchLiveFlightsUpstream() error = %v", err)
	}
	if len(data.Flights) != 1 || data.Flights[0].ID != "flight-1" {
		t.Fatalf("flights = %+v", data.Flights)
	}
	if stub.lastFlightAircraftID != "ac-1" {
		t.Fatalf("lastFlightAircraftID = %q", stub.lastFlightAircraftID)
	}
}
