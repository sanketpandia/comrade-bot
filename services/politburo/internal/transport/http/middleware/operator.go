package middleware

import (
	"net/http"
	"slices"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/transport/http/response"
)

// RequirePlatformOperator gates routes to configured operator Discord user ids.
func RequirePlatformOperator(allowedDiscordIDs []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok || claims.DsUserID == "" {
				response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "platform operator access required")
				return
			}
			if !slices.Contains(allowedDiscordIDs, claims.DsUserID) {
				response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "platform operator access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
