package va

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/routes"
)

type vaRepoStub struct {
	va  *virtualairlines.VirtualAirline
	err error
}

func (s vaRepoStub) GetByID(_ context.Context, id string) (*virtualairlines.VirtualAirline, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.va == nil || s.va.ID != id {
		return nil, nil
	}
	return s.va, nil
}

func withClaims(r *http.Request, claims auth.Claims) *http.Request {
	return r.WithContext(auth.SetClaims(r.Context(), claims))
}

func TestGetVAReturnsDetails(t *testing.T) {
	handler := &Handler{vas: vaRepoStub{va: &virtualairlines.VirtualAirline{
		ID: "va-1", Name: "Test VA", Code: "TST", IsActive: true,
	}}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VA, nil), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.GetVA(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	if data["code"] != "TST" || data["name"] != "Test VA" || data["id"] != "va-1" || data["isActive"] != true {
		t.Fatalf("data = %#v", data)
	}
}

func TestGetVAMissingVAContext(t *testing.T) {
	handler := &Handler{vas: vaRepoStub{}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VA, nil), auth.Claims{Role: "administrator"})
	handler.GetVA(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestGetVANotFound(t *testing.T) {
	handler := &Handler{vas: vaRepoStub{}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VA, nil), auth.Claims{
		PbServerID: "missing",
		Role:       "administrator",
	})
	handler.GetVA(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestGetVALoadError(t *testing.T) {
	handler := &Handler{vas: vaRepoStub{err: errors.New("db down")}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VA, nil), auth.Claims{
		PbServerID: "va-1",
		Role:       "administrator",
	})
	handler.GetVA(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
