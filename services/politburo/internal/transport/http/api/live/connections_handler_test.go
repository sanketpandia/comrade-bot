package live

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/routes"
)

type connectionsRepoStub struct {
	rows []valive.Connection
	err  error
}

func (s connectionsRepoStub) ListByVAID(context.Context, string) ([]valive.Connection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func (s connectionsRepoStub) GetByVAAndOrg(context.Context, string, string) (*valive.Connection, error) {
	return nil, nil
}

func (s connectionsRepoStub) Delete(context.Context, string, string, string) error {
	return nil
}

func TestListConnectionsRequiresVAContext(t *testing.T) {
	handler := NewConnectionsHandler(nil, nil, nil, currentUserStub{user: nil})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VALiveConnections, nil), auth.Claims{
		PbUserID: "user-1",
		Role:     "administrator",
	})
	recorder := httptest.NewRecorder()
	handler.ListConnections(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestListConnectionsReturnsPersistedRows(t *testing.T) {
	linkedAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	repo := connectionsRepoStub{rows: []valive.Connection{
		{
			VAID:               "va-1",
			LiveOrganizationID: "org-1",
			OrganizationName:   "Test VA",
			OrganizationRole:   2,
			LinkedAt:           linkedAt,
		},
	}}
	handler := &ConnectionsHandler{
		connections: repo,
		users:       currentUserStub{user: nil},
	}
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VALiveConnections, nil), auth.Claims{
		PbUserID:   "user-1",
		PbServerID: "va-1",
		Role:       "administrator",
	})
	recorder := httptest.NewRecorder()
	handler.ListConnections(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0]["id"] != "org-1" {
		t.Fatalf("data = %+v", body.Data)
	}
}
