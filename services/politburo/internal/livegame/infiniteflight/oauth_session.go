package infiniteflight

import "time"

const OAuthPendingTTL = 10 * time.Minute

// VaLiveOAuthTokenTTL is how long VA-scoped token blobs remain in Redis between refreshes.
const VaLiveOAuthTokenTTL = 30 * 24 * time.Hour

// OAuthPending is short-lived PKCE state stored before the user returns from authorize.
type OAuthPending struct {
	UserID       string `json:"user_id"`
	CodeVerifier string `json:"code_verifier"`
	VaID         string `json:"va_id,omitempty"`
	AdminScopes  bool   `json:"admin_scopes,omitempty"`
}

// OAuthTokenSession is the stored token set for a Politburo user after a successful callback.
type OAuthTokenSession struct {
	AccessToken     string    `json:"access_token"`
	RefreshToken    string    `json:"refresh_token"`
	TokenType       string    `json:"token_type"`
	Scope           string    `json:"scope"`
	AccessExpiresAt time.Time `json:"access_expires_at"`
}
