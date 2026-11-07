package va

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/membership"
	"infinite-experiment/politburo/internal/routes"

	"github.com/DATA-DOG/go-sqlmock"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestListVAMembersReturnsRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := membership.NewRepository(db)
	svc := membership.NewService(nil, repo, nil, nil)
	handler := &Handler{memberships: svc}

	joined := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`FROM public.va_user_roles vur`).
		WithArgs("va-1", true).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "if_community_id", "username", "membership_id", "role", "is_active", "callsign", "airtable_pilot_id", "joined_at", "updated_at",
		}).AddRow("user-1", "PilotOne", "display", "m-1", "proletariat", true, "TST101", nil, joined, updated))

	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.VAMembers, nil), auth.Claims{
		PbServerID: "va-1",
		Role:       "bourgeoisie",
	})
	handler.ListVAMembers(recorder, req, politburoapi.ListVAMembersParams{})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data = %#v", body["data"])
	}
}

func TestUpdateVAMemberRoleForbiddenForStaff(t *testing.T) {
	handler := &Handler{memberships: membership.NewService(nil, membership.NewRepository(mustDB(t)), nil, nil)}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodPatch, routes.VAMembers+"/user-1", bytes.NewReader([]byte(`{"role":"bourgeoisie"}`))), auth.Claims{
		PbServerID: "va-1",
		PbUserID:   "staff-1",
		Role:       "bourgeoisie",
	})
	handler.UpdateVAMemberRole(recorder, req, openapi_types.UUID([16]byte{1}))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func mustDB(t *testing.T) *sql.DB {
	t.Helper()
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
