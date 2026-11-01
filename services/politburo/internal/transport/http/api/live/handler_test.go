package live

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/routes"
)

type liveCacheStub struct {
	mu       sync.Mutex
	store    map[string][]byte
	setCalls int
}

func newLiveCacheStub() *liveCacheStub {
	return &liveCacheStub{store: map[string][]byte{}}
}

func (s *liveCacheStub) SetJSON(_ context.Context, key string, value any, _ time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.store[key] = payload
	s.setCalls++
	s.mu.Unlock()
	return nil
}

func (s *liveCacheStub) GetJSON(_ context.Context, key string, destination any) error {
	s.mu.Lock()
	payload, ok := s.store[key]
	s.mu.Unlock()
	if !ok {
		return cache.ErrMiss
	}
	return json.Unmarshal(payload, destination)
}

type orgClientStub struct {
	orgs []infiniteflight.LiveOrganization
	err  error
}

func (s orgClientStub) ListOrganizations(context.Context, string) ([]infiniteflight.LiveOrganization, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.orgs, nil
}

type currentUserStub struct {
	user *users.User
}

func (s currentUserStub) Current(context.Context, auth.Claims) (*users.User, error) {
	return s.user, nil
}

func withClaims(r *http.Request, claims auth.Claims) *http.Request {
	return r.WithContext(auth.SetClaims(r.Context(), claims))
}

func TestListOrganizationsFiltersMemberRole(t *testing.T) {
	cacheStub := newLiveCacheStub()
	token := infiniteflight.OAuthTokenSession{
		AccessToken:     "at",
		AccessExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := cacheStub.SetJSON(context.Background(), cache.KeyIFOAuthUser("user-1"), token, time.Hour); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	cacheStub.setCalls = 0

	handler := NewHandler(orgClientStub{orgs: []infiniteflight.LiveOrganization{
		{ID: "a", Name: "Alpha", Role: 0},
		{ID: "b", Name: "Bravo", Role: 1},
		{ID: "c", Name: "Charlie", Role: 2},
	}}, cacheStub, currentUserStub{user: &users.User{ID: "user-1"}})

	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveOrganizations, nil), auth.Claims{PbUserID: "user-1"})
	recorder := httptest.NewRecorder()
	handler.ListOrganizations(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body struct {
		Data []organizationItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 2 || body.Data[0].Role != 1 || body.Data[1].Role != 2 {
		t.Fatalf("data = %+v", body.Data)
	}
	if cacheStub.setCalls != 0 {
		t.Fatalf("SetJSON calls = %d, want 0", cacheStub.setCalls)
	}
}

func TestListOrganizationsOAuthRequiredWhenOnlyVATokenStored(t *testing.T) {
	cacheStub := newLiveCacheStub()
	vaToken := infiniteflight.OAuthTokenSession{
		AccessToken:     "va-at",
		AccessExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := cacheStub.SetJSON(context.Background(), cache.KeyIFOAuthVA("va-1"), vaToken, time.Hour); err != nil {
		t.Fatalf("seed va token: %v", err)
	}
	handler := NewHandler(orgClientStub{}, cacheStub, currentUserStub{user: &users.User{ID: "user-1"}})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveOrganizations, nil), auth.Claims{
		PbUserID:   "user-1",
		PbServerID: "va-1",
	})
	recorder := httptest.NewRecorder()
	handler.ListOrganizations(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (admin oauth stores VA token, not user token)", recorder.Code)
	}
}

func TestListOrganizationsOAuthRequiredWhenTokenMissing(t *testing.T) {
	handler := NewHandler(orgClientStub{}, newLiveCacheStub(), currentUserStub{user: &users.User{ID: "user-1"}})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveOrganizations, nil), auth.Claims{PbUserID: "user-1"})
	recorder := httptest.NewRecorder()
	handler.ListOrganizations(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", recorder.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"]["code"] != "IF_OAUTH_REQUIRED" {
		t.Fatalf("code = %q", body["error"]["code"])
	}
}

func TestListOrganizationsOAuthRequiredWhenTokenExpired(t *testing.T) {
	cacheStub := newLiveCacheStub()
	token := infiniteflight.OAuthTokenSession{
		AccessToken:     "at",
		AccessExpiresAt: time.Now().UTC().Add(-time.Minute),
	}
	if err := cacheStub.SetJSON(context.Background(), cache.KeyIFOAuthUser("user-1"), token, time.Hour); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	handler := NewHandler(orgClientStub{}, cacheStub, currentUserStub{user: &users.User{ID: "user-1"}})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveOrganizations, nil), auth.Claims{PbUserID: "user-1"})
	recorder := httptest.NewRecorder()
	handler.ListOrganizations(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", recorder.Code)
	}
}
