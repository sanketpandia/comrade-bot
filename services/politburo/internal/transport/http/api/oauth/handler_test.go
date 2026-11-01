package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/routes"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
)

type oauthCacheStub struct {
	mu    sync.Mutex
	store map[string][]byte
}

func newOAuthCacheStub() *oauthCacheStub {
	return &oauthCacheStub{store: map[string][]byte{}}
}

func (s *oauthCacheStub) SetJSON(_ context.Context, key string, value any, _ time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.store[key] = payload
	s.mu.Unlock()
	return nil
}

func (s *oauthCacheStub) GetDelJSON(_ context.Context, key string, destination any) error {
	s.mu.Lock()
	payload, ok := s.store[key]
	if ok {
		delete(s.store, key)
	}
	s.mu.Unlock()
	if !ok {
		return cache.ErrMiss
	}
	return json.Unmarshal(payload, destination)
}

func withClaims(r *http.Request, claims auth.Claims) *http.Request {
	return r.WithContext(auth.SetClaims(r.Context(), claims))
}

type currentUserStub struct {
	user *users.User
	err  error
}

func (s currentUserStub) Current(context.Context, auth.Claims) (*users.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func userStub(id string) currentUserStub {
	return currentUserStub{user: &users.User{ID: id}}
}

func newTestOAuthHandler(client *infiniteflight.OAuthClient, redirect string, cache tokenCache, users currentUser) *Handler {
	return NewHandler(client, redirect, cache, users, nil, nil, nil)
}

func TestGetAuthorizeURLReturnsAuthorizeLinkAndPendingSession(t *testing.T) {
	oauthClient, err := infiniteflight.NewOAuthClient("https://auth.example/v2", "ifc_test", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	cacheStub := newOAuthCacheStub()
	handler := newTestOAuthHandler(oauthClient, "http://localhost:8082"+routes.OAuthCallback, cacheStub, userStub("user-1"))

	req := withClaims(httptest.NewRequest(http.MethodGet, routes.OAuthInfiniteFlight, nil), auth.Claims{PbUserID: "user-1"})
	recorder := httptest.NewRecorder()
	handler.GetAuthorizeURL(recorder, req, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	authorizeURL, err := url.Parse(body["data"]["url"])
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	query := authorizeURL.Query()
	if query.Get("client_id") != "ifc_test" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("query = %v", query)
	}
	state := query.Get("state")
	cacheStub.mu.Lock()
	defer cacheStub.mu.Unlock()
	var pending infiniteflight.OAuthPending
	if err := json.Unmarshal(cacheStub.store[cache.KeyIFOAuthPending(state)], &pending); err != nil {
		t.Fatalf("decode pending: %v", err)
	}
	if pending.UserID != "user-1" || pending.CodeVerifier == "" {
		t.Fatalf("pending = %#v", pending)
	}
}

func TestGetAuthorizeURLUsesCurrentUserID(t *testing.T) {
	cacheStub := newOAuthCacheStub()
	handler := newTestOAuthHandler(
		mustOAuthClient(t, "https://auth.example/v2", "ifc_test", "secret"),
		"http://localhost:8082"+routes.OAuthCallback,
		cacheStub,
		userStub("user-from-discord"),
	)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.OAuthInfiniteFlight, nil), auth.Claims{
		DsUserID: "ds-1", APIKeyPresent: true,
	})
	recorder := httptest.NewRecorder()
	handler.GetAuthorizeURL(recorder, req, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	authorizeURL, err := url.Parse(body["data"]["url"])
	if err != nil {
		t.Fatalf("parse authorize url: %v", err)
	}
	var pending infiniteflight.OAuthPending
	if err := json.Unmarshal(cacheStub.store[cache.KeyIFOAuthPending(authorizeURL.Query().Get("state"))], &pending); err != nil {
		t.Fatalf("decode pending: %v", err)
	}
	if pending.UserID != "user-from-discord" {
		t.Fatalf("pending user = %q", pending.UserID)
	}
}

func TestGetAuthorizeURLCurrentUserErrors(t *testing.T) {
	client := mustOAuthClient(t, "https://auth.example/v2", "ifc_test", "secret")
	redirect := "http://localhost:8082" + routes.OAuthCallback
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"unauthorized", userapi.ErrUnauthorized, http.StatusUnauthorized},
		{"not found", userapi.ErrNotFound, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := newTestOAuthHandler(client, redirect, newOAuthCacheStub(), currentUserStub{err: tc.err})
			req := withClaims(httptest.NewRequest(http.MethodGet, routes.OAuthInfiniteFlight, nil), auth.Claims{DsUserID: "ds-1"})
			recorder := httptest.NewRecorder()
			handler.GetAuthorizeURL(recorder, req, "")
			if recorder.Code != tc.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.status)
			}
		})
	}
}

func TestCompleteCallbackStoresTokenSession(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/connect/token" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","token_type":"Bearer","scope":"openid","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	oauthClient, err := infiniteflight.NewOAuthClient(tokenServer.URL, "ifc_test", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	cacheStub := newOAuthCacheStub()
	state := "state-abc"
	_ = cacheStub.SetJSON(context.Background(), cache.KeyIFOAuthPending(state), infiniteflight.OAuthPending{
		UserID: "user-1", CodeVerifier: "verifier-xyz",
	}, infiniteflight.OAuthPendingTTL)

	handler := newTestOAuthHandler(oauthClient, "http://localhost:8082"+routes.OAuthCallback, cacheStub, nil)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.OAuthCallback+"?code=auth-code&state="+state, nil), auth.Claims{PbUserID: "user-1"})
	recorder := httptest.NewRecorder()
	handler.CompleteCallback(recorder, req, "auth-code", state)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != routes.AdminLive+"?oauth=connected" {
		t.Fatalf("location = %q", loc)
	}
	var stored infiniteflight.OAuthTokenSession
	if err := cacheStub.GetDelJSON(context.Background(), cache.KeyIFOAuthUser("user-1"), &stored); err != nil {
		t.Fatalf("load token session: %v", err)
	}
	if stored.AccessToken != "at" || stored.RefreshToken != "rt" {
		t.Fatalf("stored = %#v", stored)
	}
}

func TestCompleteCallbackStoresTokenSessionWithoutCookie(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","token_type":"Bearer","scope":"openid","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	oauthClient, err := infiniteflight.NewOAuthClient(tokenServer.URL, "ifc_test", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	cacheStub := newOAuthCacheStub()
	state := "state-no-cookie"
	_ = cacheStub.SetJSON(context.Background(), cache.KeyIFOAuthPending(state), infiniteflight.OAuthPending{
		UserID: "user-1", CodeVerifier: "verifier-xyz",
	}, infiniteflight.OAuthPendingTTL)

	handler := newTestOAuthHandler(oauthClient, "http://localhost:8082"+routes.OAuthCallback, cacheStub, nil)
	recorder := httptest.NewRecorder()
	handler.CompleteCallback(recorder, httptest.NewRequest(http.MethodGet, routes.OAuthCallback, nil), "auth-code", state)
	if loc := recorder.Header().Get("Location"); loc != routes.AdminLive+"?oauth=connected" {
		t.Fatalf("location = %q", loc)
	}
}

func TestCompleteCallbackRejectsMissingPendingState(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("token endpoint should not be called")
	}))
	defer tokenServer.Close()

	oauthClient, err := infiniteflight.NewOAuthClient(tokenServer.URL, "ifc_test", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	handler := newTestOAuthHandler(oauthClient, "http://localhost:8082/oauth/callback", newOAuthCacheStub(), nil)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.OAuthCallback+"?code=auth-code&state=missing", nil), auth.Claims{PbUserID: "user-1"})
	recorder := httptest.NewRecorder()
	handler.CompleteCallback(recorder, req, "auth-code", "missing")
	if loc := recorder.Header().Get("Location"); !strings.Contains(loc, "oauth=error") {
		t.Fatalf("location = %q", loc)
	}
}

func mustOAuthClient(t *testing.T, base, id, secret string) *infiniteflight.OAuthClient {
	c, err := infiniteflight.NewOAuthClient(base, id, secret, time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	return c
}
