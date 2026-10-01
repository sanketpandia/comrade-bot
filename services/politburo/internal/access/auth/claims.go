// Package auth holds request claims and context helpers used by middleware.
package auth

import "context"

type contextKey string

const claimsKey contextKey = "politburo.claims"

// Claims is the authenticated caller identity attached to a request.
type Claims struct {
	PbUserID      string
	Role          string
	DsUserID      string
	DsServerID    string
	PbServerID    string
	APIKeyPresent bool
}

func SetClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(Claims)
	return claims, ok
}

func (c Claims) IsMember() bool {
	return c.Role == "proletariat" || c.Role == "bourgeoisie" || c.Role == "administrator"
}

func (c Claims) IsBourgeoisie() bool {
	return c.Role == "bourgeoisie" || c.Role == "administrator"
}

func (c Claims) IsAdministrator() bool {
	return c.Role == "administrator"
}

// IsStaff is an alias for bourgeoisie-level access (includes administrator).
func (c Claims) IsStaff() bool {
	return c.IsBourgeoisie()
}

// IsAdmin is an alias for VA administrator.
func (c Claims) IsAdmin() bool {
	return c.IsAdministrator()
}
