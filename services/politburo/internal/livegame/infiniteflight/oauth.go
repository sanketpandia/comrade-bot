package infiniteflight

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultOAuthScopes are the read scopes used for dashboard connect.
const DefaultOAuthScopes = "openid profile live:organizations.read live:members.read live:aircraft.read live:schedules.read"

// OAuthClient talks to the Infinite Flight authorization server (not /public/v2).
type OAuthClient struct {
	authBaseURL  string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// OAuthTokens is the token endpoint response.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresIn    int
}

func NewOAuthClient(authBaseURL, clientID, clientSecret string, timeout time.Duration) (*OAuthClient, error) {
	authBaseURL = strings.TrimRight(strings.TrimSpace(authBaseURL), "/")
	if authBaseURL == "" {
		return nil, fmt.Errorf("OAuth auth base URL is empty")
	}
	if strings.TrimSpace(clientID) == "" {
		return nil, fmt.Errorf("OAuth client id is empty")
	}
	return &OAuthClient{
		authBaseURL:  authBaseURL,
		clientID:     strings.TrimSpace(clientID),
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: timeout},
	}, nil
}

// NewPKCEPair returns a verifier and S256 challenge for the authorize request.
func NewPKCEPair() (verifier, challenge string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate PKCE verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

// NewOAuthState returns opaque state for CSRF protection.
func NewOAuthState() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate OAuth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// AuthorizationURL builds the /connect/authorize redirect target.
func (c *OAuthClient) AuthorizationURL(redirectURI, state, codeChallenge, scopes string) (string, error) {
	if strings.TrimSpace(redirectURI) == "" {
		return "", fmt.Errorf("redirect URI is required")
	}
	if strings.TrimSpace(state) == "" {
		return "", fmt.Errorf("state is required")
	}
	if strings.TrimSpace(codeChallenge) == "" {
		return "", fmt.Errorf("code challenge is required")
	}
	if scopes == "" {
		scopes = DefaultOAuthScopes
	}
	values := url.Values{
		"response_type":         {"code"},
		"client_id":             {c.clientID},
		"redirect_uri":          {redirectURI},
		"scope":                   {scopes},
		"state":                   {state},
		"code_challenge":          {codeChallenge},
		"code_challenge_method":   {"S256"},
	}
	return c.authBaseURL + "/connect/authorize?" + values.Encode(), nil
}

// ExchangeCode trades an authorization code for tokens.
func (c *OAuthClient) ExchangeCode(ctx context.Context, redirectURI, code, codeVerifier string) (OAuthTokens, error) {
	if strings.TrimSpace(code) == "" {
		return OAuthTokens{}, fmt.Errorf("authorization code is required")
	}
	if strings.TrimSpace(codeVerifier) == "" {
		return OAuthTokens{}, fmt.Errorf("code verifier is required")
	}
	body := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"redirect_uri":  {redirectURI},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authBaseURL+"/connect/token", strings.NewReader(body.Encode()))
	if err != nil {
		return OAuthTokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return OAuthTokens{}, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return OAuthTokens{}, fmt.Errorf("decode token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return OAuthTokens{}, fmt.Errorf("token response missing access_token")
	}
	return OAuthTokens{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		TokenType:    parsed.TokenType,
		Scope:        parsed.Scope,
		ExpiresIn:    parsed.ExpiresIn,
	}, nil
}
