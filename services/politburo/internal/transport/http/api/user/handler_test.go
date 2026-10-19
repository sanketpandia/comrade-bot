package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
)

type userRepoStub struct {
	byID      *users.User
	byDiscord *users.User
	err       error
}

func (s userRepoStub) GetByID(_ context.Context, id string) (*users.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.byID == nil || s.byID.ID != id {
		return nil, nil
	}
	return s.byID, nil
}

func (s userRepoStub) GetByDiscordID(_ context.Context, discordID string) (*users.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.byDiscord == nil || s.byDiscord.DiscordID != discordID {
		return nil, nil
	}
	return s.byDiscord, nil
}

func withClaims(r *http.Request, claims auth.Claims) *http.Request {
	return r.WithContext(auth.SetClaims(r.Context(), claims))
}

func TestGetMeReturnsProfileWithRole(t *testing.T) {
	username := "pilot1"
	ifc := "PilotIFC"
	handler := &Handler{users: userRepoStub{byID: &users.User{
		ID: "user-1", DiscordID: "discord-1", IsActive: true,
		Username: &username, IFCommunityID: &ifc,
	}}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil), auth.Claims{
		PbUserID:   "user-1",
		DsUserID:   "discord-1",
		DsServerID: "guild-1",
		PbServerID: "va-1",
		Role:       "proletariat",
	})
	handler.GetMe(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	if data["id"] != "user-1" || data["discordId"] != "discord-1" || data["role"] != "proletariat" {
		t.Fatalf("data = %#v", data)
	}
	if data["vaId"] != "va-1" || data["discordServerId"] != "guild-1" {
		t.Fatalf("context fields = %#v", data)
	}
}

func TestGetMeNullRoleWhenAbsent(t *testing.T) {
	handler := &Handler{users: userRepoStub{byID: &users.User{
		ID: "user-1", DiscordID: "discord-1", IsActive: true,
	}}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil), auth.Claims{
		PbUserID: "user-1",
	})
	handler.GetMe(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, _ := body["data"].(map[string]any)
	if data["role"] != nil {
		t.Fatalf("role = %#v, want null", data["role"])
	}
}

func TestGetMeByDiscordID(t *testing.T) {
	handler := &Handler{users: userRepoStub{byDiscord: &users.User{
		ID: "user-2", DiscordID: "discord-2", IsActive: true,
	}}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil), auth.Claims{
		DsUserID: "discord-2",
		Role:     "administrator",
		APIKeyPresent: true,
	})
	handler.GetMe(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}

func TestGetMeUnauthorizedWithoutIdentity(t *testing.T) {
	handler := &Handler{users: userRepoStub{}}
	recorder := httptest.NewRecorder()
	handler.GetMe(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestGetMeNotFound(t *testing.T) {
	handler := &Handler{users: userRepoStub{}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil), auth.Claims{PbUserID: "missing"})
	handler.GetMe(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestGetMeLoadError(t *testing.T) {
	handler := &Handler{users: userRepoStub{err: errors.New("db down")}}
	recorder := httptest.NewRecorder()
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil), auth.Claims{PbUserID: "user-1"})
	handler.GetMe(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
