package ui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/routes"
	appui "infinite-experiment/politburo/internal/ui"
)

type flightsLiveStub struct {
	aircraft          []infiniteflight.LiveAircraft
	flights           []infiniteflight.LiveAircraftFlight
	flightPage        infiniteflight.LiveAircraftFlightPage
	aircraftErr       error
	flightErr         error
	lastFlightAircraftID string
	flightCalls       int
}

func (s *flightsLiveStub) ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error) {
	if s.aircraftErr != nil {
		return nil, s.aircraftErr
	}
	return s.aircraft, nil
}

func (s *flightsLiveStub) ListOrganizationSchedules(context.Context, string, string, int, int, []string) (infiniteflight.LiveSchedulePage, error) {
	return infiniteflight.LiveSchedulePage{}, nil
}

func (s *flightsLiveStub) ListAircraftSchedules(context.Context, string, string) ([]infiniteflight.LiveSchedule, error) {
	return nil, nil
}

func (s *flightsLiveStub) ListAircraftFlights(_ context.Context, _, aircraftID string, page, pageSize int) (infiniteflight.LiveAircraftFlightPage, error) {
	s.flightCalls++
	s.lastFlightAircraftID = aircraftID
	if s.flightErr != nil {
		return infiniteflight.LiveAircraftFlightPage{}, s.flightErr
	}
	if s.flightPage.TotalPages > 0 || len(s.flightPage.Items) > 0 {
		return s.flightPage, nil
	}
	return infiniteflight.LiveAircraftFlightPage{
		Items:      s.flights,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: len(s.flights),
		TotalPages: 1,
	}, nil
}

func TestLiveFlightsListForbiddenForNonAdmin(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &flightsLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{Role: "proletariat", PbServerID: "va-1"})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestLiveFlightsListMissingAircraftPrompt(t *testing.T) {
	live := &flightsLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1", Registration: "N682XL", FleetRank: 1}},
	}
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        live,
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
		Liveries:    gameliveries.NewLookup(nil),
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if live.flightCalls != 0 {
		t.Fatalf("flightCalls = %d, want 0", live.flightCalls)
	}
	if !strings.Contains(body, "Select an aircraft to load flight history.") {
		t.Fatalf("missing prompt in %s", body)
	}
}

func TestLiveFlightsListRendersTable(t *testing.T) {
	departure, _ := time.Parse(time.RFC3339, "2026-07-03T18:42:00Z")
	arrival, _ := time.Parse(time.RFC3339, "2026-07-03T23:51:00Z")
	username := "PilotOne"
	live := &flightsLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{
			ID:           "ac-persist-1",
			AircraftID:   "content-1",
			Registration: "N682XL",
			FleetRank:    1,
		}},
		flights: []infiniteflight.LiveAircraftFlight{{
			ID:                 "flight-1",
			AircraftID:         "ac-persist-1",
			Callsign:           "N682XL",
			OriginIcao:         strPtr("KLAX"),
			DestinationIcao:    strPtr("KJFK"),
			ActualDepartureUtc: &departure,
			ActualArrivalUtc:   &arrival,
			Status:             1,
			FlightTimeMinutes:    309,
			Crew: []infiniteflight.LiveAircraftFlightCrewMember{{
				UserID:   "user-1",
				Username: &username,
			}},
		}},
	}
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        live,
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
		Liveries:    gameliveries.NewLookup(nil),
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-persist-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"1 flight",
		"N682XL",
		"KLAX → KJFK",
		"Completed",
		"5h 9m",
		"PilotOne",
		`data-flight-id="flight-1"`,
		`data-aircraft-id="ac-persist-1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if live.lastFlightAircraftID != "ac-persist-1" {
		t.Fatalf("lastFlightAircraftID = %q", live.lastFlightAircraftID)
	}
}

func TestLiveFlightsListEmpty(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live: &flightsLiveStub{
			aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1", FleetRank: 1}},
		},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "No flights in the log for this aircraft.") {
		t.Fatalf("missing empty message in %s", body)
	}
}

func TestLiveFlightsListOAuthRequired(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &flightsLiveStub{},
		Tokens:      fleetTokensStub{err: valive.ErrOAuthRequired},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if !strings.Contains(rec.Body.String(), "Connect Infinite Flight") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLiveFlightsListUnlinkedOrg(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &flightsLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: nil},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if !strings.Contains(rec.Body.String(), "not linked") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLiveFlightsListAircraftUpstreamFailure(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &flightsLiveStub{aircraftErr: errors.New("aircraft down")},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if !strings.Contains(rec.Body.String(), "Could not load aircraft") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLiveFlightsListFlightsUpstreamFailure(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live: &flightsLiveStub{
			aircraft:  []infiniteflight.LiveAircraft{{ID: "ac-1"}},
			flightErr: errors.New("flights down"),
		},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if !strings.Contains(rec.Body.String(), "Could not load flights") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLiveFlightsListForwardsAircraftFilter(t *testing.T) {
	live := &flightsLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{ID: "ac-filter-1"}},
	}
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        live,
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-filter-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if live.lastFlightAircraftID != "ac-filter-1" {
		t.Fatalf("lastFlightAircraftID = %q", live.lastFlightAircraftID)
	}
}

func TestLiveFlightsListTruncationNote(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live: &flightsLiveStub{
			aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1"}},
			flightPage: infiniteflight.LiveAircraftFlightPage{
				Items:      []infiniteflight.LiveAircraftFlight{{ID: "f1", AircraftID: "ac-1", Status: 1}},
				Page:       10,
				PageSize:   100,
				TotalCount: 1001,
				TotalPages: 11,
			},
		},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlightsList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveFlightsListPartial(rec, req)
	if !strings.Contains(rec.Body.String(), "Showing the first 1000 flights.") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLiveModeRendersFlightsTab(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, false, nil, nil)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveFlights, nil), auth.Claims{Role: "administrator"})
	rec := httptest.NewRecorder()
	handler.LiveMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`href="/live/flights" aria-current="page"`,
		`id="live-flights"`,
		`id="live-flights-row-menu"`,
		`/static/js/live/flights.mjs`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func strPtr(s string) *string {
	return &s
}
