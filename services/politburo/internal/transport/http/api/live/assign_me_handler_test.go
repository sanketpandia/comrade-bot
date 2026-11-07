package live

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/webhooks/features/livemodeassignment"
)

type assignCaptainStub struct {
	result valive.AssignCaptainResult
	err    error
}

func (s *assignCaptainStub) AssignCaptain(context.Context, valive.AssignCaptainInput) (valive.AssignCaptainResult, error) {
	return s.result, s.err
}

func (s *assignCaptainStub) ResolveAircraftIDs(context.Context, string, string, string) ([]string, map[string]string, error) {
	return []string{"ac-1"}, map[string]string{"ac-1": "PH-ZNB"}, nil
}

type assignmentNotifierStub struct {
	err error
}

func (s *assignmentNotifierStub) Notify(context.Context, string, livemodeassignment.Event) error {
	return s.err
}

func TestAssignVALiveScheduleToMeLiveDisabled(t *testing.T) {
	ifID := "if-user-1"
	handler := &ConnectionsHandler{
		users: currentUserStub{user: &users.User{IFApiID: &ifID}},
	}
	body, _ := json.Marshal(map[string]string{"registration": "PH-ZNB", "route": "EHAM-CYYZ"})
	req := withClaims(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)), auth.Claims{
		PbUserID:   "user-1",
		PbServerID: "va-1",
		Role:       "proletariat",
	})
	rec := httptest.NewRecorder()
	handler.AssignVALiveScheduleToMe(rec, req, testUUID(t, testOrgID))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
}

func TestAssignVALiveScheduleToMeNotifierFailureStillOK(t *testing.T) {
	ifID := "if-user-1"
	handler := &ConnectionsHandler{
		assignCaptain: &assignCaptainStub{
			result: valive.AssignCaptainResult{
				ScheduleID:      "sched-1",
				OrganizationID:  testOrgID,
				Registration:    "PH-ZNB",
				OriginIcao:      "EHAM",
				DestinationIcao: "CYYZ",
			},
		},
		assignmentNotifier: &assignmentNotifierStub{err: errors.New("discord down")},
		users:              currentUserStub{user: &users.User{IFApiID: &ifID, DiscordID: "discord-1", Username: strPtr("PilotOne")}},
	}
	body, _ := json.Marshal(map[string]string{"registration": "PH-ZNB", "route": "EHAM-CYYZ"})
	req := withClaims(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)), auth.Claims{
		PbUserID:   "user-1",
		PbServerID: "va-1",
		Role:       "proletariat",
	})
	rec := httptest.NewRecorder()
	handler.AssignVALiveScheduleToMe(rec, req, testUUID(t, testOrgID))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
}

func strPtr(s string) *string { return &s }

func TestAssignVALiveScheduleToMeMissingIFUser(t *testing.T) {
	handler := &ConnectionsHandler{
		assignCaptain: valive.NewAssignCaptainService(nil, nil, nil),
		users:         currentUserStub{user: &users.User{}},
	}
	body, _ := json.Marshal(map[string]string{"registration": "PH-ZNB", "route": "EHAM-CYYZ"})
	req := withClaims(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)), auth.Claims{
		PbUserID:   "user-1",
		PbServerID: "va-1",
		Role:       "proletariat",
	})
	rec := httptest.NewRecorder()
	handler.AssignVALiveScheduleToMe(rec, req, testUUID(t, testOrgID))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
}
