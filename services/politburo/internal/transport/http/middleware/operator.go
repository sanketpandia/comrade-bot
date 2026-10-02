package middleware

import (
	"log/slog"
	"net/http"
	"slices"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/transport/http/response"
)

// RequirePlatformOperator gates routes to configured operator Discord user ids.
func RequirePlatformOperator(allowedDiscordIDs []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok || claims.DsUserID == "" {
				logAPIAccessDenied(r, "missing_caller_discord_id", "error_code", "FORBIDDEN", "has_claims", ok)
				response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "platform operator access required")
				return
			}
			if !slices.Contains(allowedDiscordIDs, claims.DsUserID) {
				logAPIAccessDenied(r, "not_platform_operator", "error_code", "FORBIDDEN", "discord_user_id", claims.DsUserID)
				response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "platform operator access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func logAPIAccessDenied(r *http.Request, reason string, extra ...any) {
	args := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"reason", reason,
	}
	if requestID := chimiddleware.GetReqID(r.Context()); requestID != "" {
		args = append(args, "request_id", requestID)
	}
	args = append(args, extra...)
	slog.Info("api_access_denied", args...)
}
