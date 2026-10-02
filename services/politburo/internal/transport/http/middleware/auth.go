package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/transport/http/response"
)

const (
	APIKeyHeader          = "X-API-Key"
	DiscordUserIDHeader   = "X-Discord-User-Id"
	DiscordServerIDHeader = "X-Discord-Server-Id"
)

// APIKeyLookup validates an API key. Implementations are wired when key
// storage exists; a nil lookup leaves APIKeyAuth as a pass-through scaffold.
type APIKeyLookup interface {
	Lookup(r *http.Request, apiKey string) (auth.Claims, bool, error)
}

// APIKeyAuth requires a valid API key for /api/v1 paths when a lookup is
// configured. Health, metrics, and UI paths pass through. With a nil lookup
// the middleware is a no-op scaffold.
func APIKeyAuth(lookup APIKeyLookup) func(http.Handler) http.Handler {
	return AuthenticateAPI(lookup, nil)
}

// AuthenticateAPI authenticates /api/v1 requests. Game read paths accept a
// session cookie or an API key (cookie first). Other /api/v1 paths, including
// signed-link minting, require an API key. Health, metrics, and UI pass through.
func AuthenticateAPI(lookup APIKeyLookup, sessions SessionLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if lookup == nil || !strings.HasPrefix(r.URL.Path, "/api/v1") {
				next.ServeHTTP(w, r)
				return
			}

			if sessions != nil && isSessionAPIPath(r.URL.Path) {
				claims, ok, err := sessions.Lookup(r)
				if err != nil {
					response.WriteError(w, http.StatusInternalServerError, "AUTH_LOOKUP_FAILED", "authentication lookup failed")
					return
				}
				if ok {
					next.ServeHTTP(w, r.WithContext(auth.SetClaims(r.Context(), claims)))
					return
				}
			}

			apiKey := strings.TrimSpace(r.Header.Get(APIKeyHeader))
			if apiKey == "" {
				logAPIAuthFailure(r, "missing_api_key")
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
				return
			}
			claims, ok, err := lookup.Lookup(r, apiKey)
			if err != nil {
				logAPIAuthFailure(r, "api_key_lookup_error", "lookup_error", err.Error())
				response.WriteError(w, http.StatusInternalServerError, "AUTH_LOOKUP_FAILED", "authentication lookup failed")
				return
			}
			if !ok {
				logAPIAuthFailure(r, "invalid_api_key")
				response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
				return
			}
			claims.APIKeyPresent = true
			claims.DsUserID = strings.TrimSpace(r.Header.Get(DiscordUserIDHeader))
			claims.DsServerID = strings.TrimSpace(r.Header.Get(DiscordServerIDHeader))
			next.ServeHTTP(w, r.WithContext(auth.SetClaims(r.Context(), claims)))
		})
	}
}

func isGameAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/v1/game/")
}

func isSessionAPIPath(path string) bool {
	return isGameAPIPath(path) || strings.HasPrefix(path, "/api/v1/operator/")
}

// RequireDiscordBotContext requires Discord user and server headers after API
// auth has succeeded. Apply to bot registration-style route groups.
func RequireDiscordBotContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hasUser := strings.TrimSpace(r.Header.Get(DiscordUserIDHeader)) != ""
			hasServer := strings.TrimSpace(r.Header.Get(DiscordServerIDHeader)) != ""
			if !hasUser || !hasServer {
				args := []any{
					"method", r.Method,
					"path", r.URL.Path,
					"reason", "missing_discord_context",
					"error_code", "MISSING_DISCORD_CONTEXT",
					"has_discord_user_id", hasUser,
					"has_discord_server_id", hasServer,
				}
				if requestID := chimiddleware.GetReqID(r.Context()); requestID != "" {
					args = append(args, "request_id", requestID)
				}
				slog.Info("api_access_denied", args...)
				response.WriteError(w, http.StatusForbidden, "MISSING_DISCORD_CONTEXT",
					"Missing required Discord context headers: X-Discord-User-Id and X-Discord-Server-Id")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func logAPIAuthFailure(r *http.Request, reason string, extra ...any) {
	args := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"reason", reason,
	}
	args = append(args, extra...)
	if requestID := chimiddleware.GetReqID(r.Context()); requestID != "" {
		args = append(args, "request_id", requestID)
	}
	slog.Info("api_auth_failed", args...)
}
