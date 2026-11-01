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

type schedulesLiveStub struct {
	aircraft     []infiniteflight.LiveAircraft
	schedules    []infiniteflight.LiveSchedule
	schedulePage infiniteflight.LiveSchedulePage
	aircraftErr  error
	scheduleErr  error
	lastOrgScheduleAircraftIDs []string
	lastAircraftScheduleID     string
	orgScheduleCalls             int
	aircraftScheduleCalls        int
}

func (s *schedulesLiveStub) ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error) {
	if s.aircraftErr != nil {
		return nil, s.aircraftErr
	}
	return s.aircraft, nil
}

func (s *schedulesLiveStub) ListOrganizationSchedules(_ context.Context, _, _ string, _, _ int, aircraftIDs []string) (infiniteflight.LiveSchedulePage, error) {
	s.orgScheduleCalls++
	s.lastOrgScheduleAircraftIDs = append([]string(nil), aircraftIDs...)
	if s.scheduleErr != nil {
		return infiniteflight.LiveSchedulePage{}, s.scheduleErr
	}
	if s.schedulePage.TotalPages > 0 || len(s.schedulePage.Items) > 0 {
		return s.schedulePage, nil
	}
	return infiniteflight.LiveSchedulePage{
		Items:      s.schedules,
		Page:       1,
		PageSize:   100,
		TotalCount: len(s.schedules),
		TotalPages: 1,
	}, nil
}

func (s *schedulesLiveStub) ListAircraftSchedules(_ context.Context, _, aircraftID string) ([]infiniteflight.LiveSchedule, error) {
	s.aircraftScheduleCalls++
	s.lastAircraftScheduleID = aircraftID
	if s.scheduleErr != nil {
		return nil, s.scheduleErr
	}
	return s.schedules, nil
}

func (s *schedulesLiveStub) ListAircraftFlights(context.Context, string, string, int, int) (infiniteflight.LiveAircraftFlightPage, error) {
	return infiniteflight.LiveAircraftFlightPage{}, nil
}

func TestLiveSchedulesListForbiddenForProletariat(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{Role: "proletariat", PbServerID: "va-1"})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestLiveSchedulesListAllowsBourgeoisie(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{aircraft: []infiniteflight.LiveAircraft{{ID: "ac-1"}}},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{LiveOrganizationID: "org-1"}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1&aircraftId=ac-1", nil), auth.Claims{
		Role:       "bourgeoisie",
		PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestLiveSchedulesListRendersTable(t *testing.T) {
	departure, _ := time.Parse(time.RFC3339, "2026-07-03T14:00:00Z")
	live := &schedulesLiveStub{
		aircraft: []infiniteflight.LiveAircraft{{
			ID:           "ac-persist-1",
			AircraftID:   "content-1",
			Registration: "N682XL",
			FleetRank:    1,
		}},
		schedules: []infiniteflight.LiveSchedule{{
			ID:                    "sched-1",
			Status:                1,
			Callsign:              "PJX421",
			AircraftID:            "ac-persist-1",
			OriginIcao:            "KJFK",
			DestinationIcao:       "KLAX",
			ScheduledDepartureUtc: departure,
		}},
	}
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        live,
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{LiveOrganizationID: "org-1"}},
		Liveries:    gameliveries.NewStaticLookup(nil, nil),
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"1 schedule",
		"N682XL",
		"PJX421",
		"KJFK → KLAX",
		"Scheduled",
		`id="live-schedules-aircraft-filter"`,
		`data-schedule-id="sched-1"`,
		`data-aircraft-id="ac-persist-1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestLiveSchedulesListEmpty(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "No schedules") {
		t.Fatalf("missing empty label: %s", body)
	}
}

func TestLiveSchedulesListOAuthRequired(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{},
		Tokens:      fleetTokensStub{err: valive.ErrOAuthRequired},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Connect Infinite Flight") {
		t.Fatalf("missing oauth message: %s", body)
	}
}

func TestLiveSchedulesListUnlinkedOrg(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: nil},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "not linked") {
		t.Fatalf("missing unlink message: %s", body)
	}
}

func TestLiveSchedulesListScheduleUpstreamFailure(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{scheduleErr: errors.New("schedules down")},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Could not load schedules") {
		t.Fatalf("missing schedule error: %s", body)
	}
}

func TestLiveSchedulesListAircraftUpstreamFailure(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        &schedulesLiveStub{aircraftErr: errors.New("aircraft down")},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Could not load aircraft") {
		t.Fatalf("missing aircraft error: %s", body)
	}
}

func TestLiveSchedulesListForwardsAircraftFilter(t *testing.T) {
	live := &schedulesLiveStub{}
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        live,
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1&aircraftId=ac-filter-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if live.orgScheduleCalls != 0 {
		t.Fatalf("org schedule calls = %d, want 0 when filtered to one aircraft", live.orgScheduleCalls)
	}
	if live.aircraftScheduleCalls != 1 || live.lastAircraftScheduleID != "ac-filter-1" {
		t.Fatalf("aircraft schedule calls = %d id = %q", live.aircraftScheduleCalls, live.lastAircraftScheduleID)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="ac-filter-1" selected`) {
		t.Fatalf("missing selected unknown aircraft filter: %s", body)
	}
}

func TestLiveSchedulesListTruncationNote(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live: &schedulesLiveStub{
			schedulePage: infiniteflight.LiveSchedulePage{
				Items:      []infiniteflight.LiveSchedule{{ID: "s1", AircraftID: "a1", Status: 1}},
				Page:       1,
				PageSize:   100,
				TotalCount: 2000,
				TotalPages: 20,
			},
		},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedulesList+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveSchedulesListPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "first 1000 schedules") {
		t.Fatalf("missing truncation note: %s", body)
	}
}

func TestLiveModeRendersSchedulesTab(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, false, nil, nil)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveSchedules, nil), auth.Claims{Role: "administrator"})
	rec := httptest.NewRecorder()
	handler.LiveMode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`href="/live/schedules" aria-current="page"`,
		`id="live-schedules"`,
		`id="live-schedules-row-menu"`,
		`/static/js/live/schedules.mjs`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}
