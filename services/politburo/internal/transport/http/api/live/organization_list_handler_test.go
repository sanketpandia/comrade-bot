package live

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/livegame/valive"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
)

const (
	testOrgID  = "11111111-1111-1111-1111-111111111111"
	testAcID   = "22222222-2222-2222-2222-222222222222"
	testAcTwo  = "33333333-3333-3333-3333-333333333333"
	testAcMiss = "44444444-4444-4444-4444-444444444444"
	testFilter = "55555555-5555-5555-5555-555555555555"
)

type liveDataStub struct {
	aircraft       []infiniteflight.LiveAircraft
	schedPage      infiniteflight.LiveSchedulePage
	flightPage     infiniteflight.LiveAircraftFlightPage
	aircraftErr    error
	schedErr       error
	flightErr      error
	lastSchedPage  int
	lastSchedSize  int
	lastSchedIDs   []string
	lastFlightPage int
	lastFlightSize int
}

func (s *liveDataStub) GetOrganization(context.Context, string, string) (infiniteflight.LiveOrganization, error) {
	return infiniteflight.LiveOrganization{}, nil
}

func (s *liveDataStub) ListOrganizationAircraft(context.Context, string, string) ([]infiniteflight.LiveAircraft, error) {
	if s.aircraftErr != nil {
		return nil, s.aircraftErr
	}
	return s.aircraft, nil
}

func (s *liveDataStub) ListOrganizationSchedules(_ context.Context, _, _ string, page, pageSize int, aircraftIDs []string) (infiniteflight.LiveSchedulePage, error) {
	s.lastSchedPage = page
	s.lastSchedSize = pageSize
	s.lastSchedIDs = append([]string(nil), aircraftIDs...)
	if s.schedErr != nil {
		return infiniteflight.LiveSchedulePage{}, s.schedErr
	}
	return s.schedPage, nil
}

func (s *liveDataStub) ListAircraftFlights(_ context.Context, _, _ string, page, pageSize int) (infiniteflight.LiveAircraftFlightPage, error) {
	s.lastFlightPage = page
	s.lastFlightSize = pageSize
	if s.flightErr != nil {
		return infiniteflight.LiveAircraftFlightPage{}, s.flightErr
	}
	return s.flightPage, nil
}

type orgConnectionsRepoStub struct {
	conn *valive.Connection
}

func (s orgConnectionsRepoStub) ListByVAID(context.Context, string) ([]valive.Connection, error) {
	return nil, nil
}

func (s orgConnectionsRepoStub) GetByVAAndOrg(context.Context, string, string) (*valive.Connection, error) {
	return s.conn, nil
}

func (s orgConnectionsRepoStub) Delete(context.Context, string, string, string) error {
	return nil
}

type tokenStub struct {
	token string
	err   error
}

func (s tokenStub) AccessToken(context.Context, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.token, nil
}

func staffClaims() auth.Claims {
	return auth.Claims{
		PbUserID:   "user-1",
		PbServerID: "va-1",
		Role:       "bourgeoisie",
	}
}

func linkedHandler(live *liveDataStub) *ConnectionsHandler {
	return &ConnectionsHandler{
		connections: orgConnectionsRepoStub{conn: &valive.Connection{VAID: "va-1", LiveOrganizationID: testOrgID}},
		live:        live,
		liveData:    live,
		tokens:      tokenStub{token: "tok"},
		users:       currentUserStub{user: nil},
	}
}

func TestListOrganizationAircraftReturnsSortedFleet(t *testing.T) {
	live := &liveDataStub{aircraft: []infiniteflight.LiveAircraft{
		{ID: testAcTwo, AircraftID: "type-2", OrganizationID: testOrgID, FleetRank: 2, CreatedAt: time.Now()},
		{ID: testAcID, AircraftID: "type-1", OrganizationID: testOrgID, FleetRank: 1, CreatedAt: time.Now()},
	}}
	handler := linkedHandler(live)
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/live/organizations/"+testOrgID+"/aircraft", nil), staffClaims())
	rec := httptest.NewRecorder()
	handler.ListOrganizationAircraft(rec, req, testUUID(t, testOrgID))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 2 || body.Data[0]["id"] != testAcID {
		t.Fatalf("data = %#v, want %s first", body.Data, testAcID)
	}
}

func TestListOrganizationSchedulesPassesPaging(t *testing.T) {
	live := &liveDataStub{schedPage: infiniteflight.LiveSchedulePage{
		Items:      []infiniteflight.LiveSchedule{{ID: "s1", AircraftID: testAcID, OrganizationID: testOrgID, ScheduledDepartureUtc: time.Now()}},
		Page:       2,
		PageSize:   50,
		TotalCount: 120,
		TotalPages: 3,
	}}
	handler := linkedHandler(live)
	page := 2
	pageSize := 50
	filter := testUUID(t, testFilter)
	params := politburoapi.ListVALiveOrganizationSchedulesParams{
		Page:       &page,
		PageSize:   &pageSize,
		AircraftId: &[]openapi_types.UUID{filter},
	}
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/live/organizations/"+testOrgID+"/schedules", nil), staffClaims())
	rec := httptest.NewRecorder()
	handler.ListOrganizationSchedules(rec, req, testUUID(t, testOrgID), params)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if live.lastSchedPage != 2 || live.lastSchedSize != 50 {
		t.Fatalf("page passthrough = %d/%d", live.lastSchedPage, live.lastSchedSize)
	}
	if len(live.lastSchedIDs) != 1 || live.lastSchedIDs[0] != testFilter {
		t.Fatalf("aircraft filter = %#v", live.lastSchedIDs)
	}
}

func TestListOrganizationSchedulesOAuthRequired(t *testing.T) {
	handler := &ConnectionsHandler{
		connections: orgConnectionsRepoStub{conn: &valive.Connection{}},
		liveData:    &liveDataStub{},
		tokens:      tokenStub{err: valive.ErrOAuthRequired},
	}
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/live/organizations/"+testOrgID+"/schedules", nil), staffClaims())
	rec := httptest.NewRecorder()
	handler.ListOrganizationSchedules(rec, req, testUUID(t, testOrgID), politburoapi.ListVALiveOrganizationSchedulesParams{})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestListAircraftFlightsRejectsUnknownAircraft(t *testing.T) {
	live := &liveDataStub{aircraft: []infiniteflight.LiveAircraft{{ID: testAcID}}}
	handler := linkedHandler(live)
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/live/organizations/"+testOrgID+"/aircraft/"+testAcMiss+"/flights", nil), staffClaims())
	rec := httptest.NewRecorder()
	handler.ListAircraftFlights(rec, req, testUUID(t, testOrgID), testUUID(t, testAcMiss), politburoapi.ListVALiveAircraftFlightsParams{})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestListAircraftFlightsReturnsPage(t *testing.T) {
	live := &liveDataStub{
		aircraft: []infiniteflight.LiveAircraft{{ID: testAcID}},
		flightPage: infiniteflight.LiveAircraftFlightPage{
			Items:      []infiniteflight.LiveAircraftFlight{{ID: "f1", AircraftID: testAcID, OrganizationID: testOrgID, Crew: []infiniteflight.LiveAircraftFlightCrewMember{{UserID: "u1"}}}},
			Page:       1,
			PageSize:   25,
			TotalCount: 1,
			TotalPages: 1,
		},
	}
	handler := linkedHandler(live)
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/va/live/organizations/"+testOrgID+"/aircraft/"+testAcID+"/flights", nil), staffClaims())
	rec := httptest.NewRecorder()
	handler.ListAircraftFlights(rec, req, testUUID(t, testOrgID), testUUID(t, testAcID), politburoapi.ListVALiveAircraftFlightsParams{})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("data len = %d", len(body.Data))
	}
}

func testUUID(t *testing.T, s string) openapi_types.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", s, err)
	}
	return openapi_types.UUID(id)
}
