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

type fleetConnectionsStub struct {
	conn *valive.Connection
	err  error
}

func (s fleetConnectionsStub) GetByVAAndOrg(context.Context, string, string) (*valive.Connection, error) {
	return s.conn, s.err
}

type fleetTokensStub struct {
	token string
	err   error
}

func (s fleetTokensStub) AccessToken(context.Context, string) (string, error) {
	return s.token, s.err
}

type fleetLiveStub struct {
	rows         []infiniteflight.LiveAircraft
	schedules    []infiniteflight.LiveSchedule
	schedulePage infiniteflight.LiveSchedulePage
	scheduleErr  error
	err          error
}

func (s fleetLiveStub) ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error) {
	return s.rows, s.err
}

func (s fleetLiveStub) ListOrganizationSchedules(context.Context, string, string, int, int, []string) (infiniteflight.LiveSchedulePage, error) {
	if s.scheduleErr != nil {
		return infiniteflight.LiveSchedulePage{}, s.scheduleErr
	}
	if s.err != nil {
		return infiniteflight.LiveSchedulePage{}, s.err
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

func (s fleetLiveStub) ListAircraftSchedules(context.Context, string, string) ([]infiniteflight.LiveSchedule, error) {
	if s.scheduleErr != nil {
		return nil, s.scheduleErr
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.schedules, nil
}

func (s fleetLiveStub) ListAircraftFlights(context.Context, string, string, int, int) (infiniteflight.LiveAircraftFlightPage, error) {
	return infiniteflight.LiveAircraftFlightPage{}, nil
}

func newLiveFleetHandler(t *testing.T, deps *LiveFleetDeps) *Handler {
	t.Helper()
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	return NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, false, nil, deps)
}

func TestLiveAircraftFleetForbiddenForProletariat(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        fleetLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{LiveOrganizationID: "org-1"}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{Role: "proletariat", PbServerID: "va-1"})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestLiveAircraftFleetAllowsBourgeoisie(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        fleetLiveStub{rows: []infiniteflight.LiveAircraft{{ID: "ac-1", Registration: "N1"}}},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{LiveOrganizationID: "org-1"}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role:       "bourgeoisie",
		PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestLiveAircraftFleetRendersTable(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-07-03T12:00:00Z")
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live: fleetLiveStub{rows: []infiniteflight.LiveAircraft{{
			ID:                "persist-1",
			AircraftID:        "content-1",
			Registration:      "N682XL",
			Status:            0,
			Visibility:        1,
			FleetRank:         1,
			IsFleetActiveSlot: true,
			CreatedAt:         created,
		}}},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{LiveOrganizationID: "org-1"}},
		Liveries:    gameliveries.NewStaticLookup(nil, nil),
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"1 aircraft",
		"N682XL",
		"badge--success",
		"Active",
		"Visible",
		"Active slot",
		`class="data-table"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestLiveAircraftFleetEmptyList(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        fleetLiveStub{rows: nil},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "No aircraft") {
		t.Fatalf("missing empty label: %s", body)
	}
}

func TestLiveAircraftFleetOAuthRequired(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        fleetLiveStub{},
		Tokens:      fleetTokensStub{err: valive.ErrOAuthRequired},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Connect Infinite Flight") {
		t.Fatalf("missing oauth message: %s", body)
	}
}

func TestLiveAircraftFleetUnlinkedOrg(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        fleetLiveStub{},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: nil},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "not linked") {
		t.Fatalf("missing unlink message: %s", body)
	}
}

func TestLiveAircraftFleetUpstreamFailure(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live:        fleetLiveStub{err: errors.New("upstream down")},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Could not load aircraft") {
		t.Fatalf("missing upstream error: %s", body)
	}
}

func TestLiveAircraftFleetRowHasPersistentID(t *testing.T) {
	handler := newLiveFleetHandler(t, &LiveFleetDeps{
		Live: fleetLiveStub{rows: []infiniteflight.LiveAircraft{{
			ID:           "persist-abc",
			AircraftID:   "content-xyz",
			Registration: "PH-BVA",
			Status:       0,
			Visibility:   1,
			FleetRank:    2,
		}}},
		Tokens:      fleetTokensStub{token: "tok"},
		Connections: fleetConnectionsStub{conn: &valive.Connection{}},
	})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraftFleet+"?organizationId=org-1", nil), auth.Claims{
		Role: "administrator", PbServerID: "va-1",
	})
	rec := httptest.NewRecorder()
	handler.LiveAircraftFleetPartial(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `data-aircraft-id="persist-abc"`) {
		t.Fatalf("missing aircraft id on row: %s", body)
	}
}
