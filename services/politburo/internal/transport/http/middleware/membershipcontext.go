package middleware

import (
	"context"
	"net/http"

	"infinite-experiment/politburo/internal/auth"
)

type MembershipResolver interface {
	ResolveMembership(ctx context.Context, discordUserID, discordServerID string) (vaID, role string, ok bool, err error)
}

// EnrichDiscordMembership sets PbServerID and Role when the guild is a VA and the user is a member.
func EnrichDiscordMembership(resolver MembershipResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver == nil {
				next.ServeHTTP(w, r)
				return
			}
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok || claims.DsUserID == "" || claims.DsServerID == "" {
				next.ServeHTTP(w, r)
				return
			}
			vaID, role, found, err := resolver.ResolveMembership(r.Context(), claims.DsUserID, claims.DsServerID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if found {
				claims.PbServerID = vaID
				claims.Role = role
				r = r.WithContext(auth.SetClaims(r.Context(), claims))
			}
			next.ServeHTTP(w, r)
		})
	}
}
