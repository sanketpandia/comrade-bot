package infiniteflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestAuthorizationURLIncludesPKCEAndCallback(t *testing.T) {
	client, err := NewOAuthClient("https://api.infiniteflight.com/auth/v2", "ifc_demo", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	_, challenge, err := NewPKCEPair()
	if err != nil {
		t.Fatalf("NewPKCEPair() error = %v", err)
	}
	authorizeURL, err := client.AuthorizationURL("http://localhost:8082/oauth/callback", "state-1", challenge, DefaultOAuthScopes)
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}
	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	query := parsed.Query()
	if query.Get("response_type") != "code" {
		t.Fatalf("response_type = %q", query.Get("response_type"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q", query.Get("code_challenge_method"))
	}
	if query.Get("redirect_uri") != "http://localhost:8082/oauth/callback" {
		t.Fatalf("redirect_uri = %q", query.Get("redirect_uri"))
	}
}

func TestExchangeCodePostsToTokenEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":60,"token_type":"Bearer","scope":"openid"}`))
	}))
	defer server.Close()

	client, err := NewOAuthClient(server.URL, "ifc_demo", "secret", time.Second)
	if err != nil {
		t.Fatalf("NewOAuthClient() error = %v", err)
	}
	tokens, err := client.ExchangeCode(context.Background(), "http://localhost:8082/oauth/callback", "code-1", "verifier-1")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/connect/token" {
		t.Fatalf("token request = %s %s", gotMethod, gotPath)
	}
	if tokens.AccessToken != "access" || tokens.RefreshToken != "refresh" {
		t.Fatalf("tokens = %#v", tokens)
	}
}
