package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
)

func TestAPIKeyAuthPassThroughWhenLookupNil(t *testing.T) {
	called := false
	handler := APIKeyAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/x", nil))
	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("pass-through failed: called=%v status=%d", called, recorder.Code)
	}
}

func TestAPIKeyAuthSkipsNonAPIPaths(t *testing.T) {
	lookup := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("lookup should not run for non-api paths")
		return auth.Claims{}, false, nil
	})
	called := false
	handler := APIKeyAuth(lookup)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/status", nil))
	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("health pass-through failed: called=%v status=%d", called, recorder.Code)
	}
}

func TestAPIKeyAuthRequiresKeyOnAPIPaths(t *testing.T) {
	lookup := apiKeyLookupFunc(func(_ *http.Request, key string) (auth.Claims, bool, error) {
		return auth.Claims{}, key == "good", nil
	})
	handler := APIKeyAuth(lookup)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/signed-link", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", nil)
	req.Header.Set(APIKeyHeader, "good")
	req.Header.Set(DiscordUserIDHeader, "user")
	req.Header.Set(DiscordServerIDHeader, "server")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPIAcceptsSessionCookieOnGamePaths(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key lookup should not run when session is valid")
		return auth.Claims{}, false, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{PbUserID: "user-1"}, true, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || claims.PbUserID != "user-1" || claims.APIKeyPresent {
			t.Fatalf("claims = %#v ok=%v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPICookieWinsOverAPIKeyOnGamePaths(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key must not be used when a session cookie authenticates")
		return auth.Claims{PbUserID: "key-user"}, true, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{PbUserID: "cookie-user"}, true, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.ClaimsFromContext(r.Context())
		if claims.PbUserID != "cookie-user" || claims.APIKeyPresent {
			t.Fatalf("claims = %#v", claims)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
	req.Header.Set(APIKeyHeader, "good")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPIFallsBackToAPIKeyOnGamePaths(t *testing.T) {
	keys := apiKeyLookupFunc(func(_ *http.Request, key string) (auth.Claims, bool, error) {
		return auth.Claims{}, key == "good", nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{}, false, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("public active flights status = %d, want 204", recorder.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil)
	req.Header.Set(APIKeyHeader, "good")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flight/active/c34118e7-cbdd-4e22-8751-0cda93e41d75", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("public active flight status = %d, want 204", recorder.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/game/flight/active/c34118e7-cbdd-4e22-8751-0cda93e41d75", nil)
	req.Header.Set(APIKeyHeader, "good")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("active flight with key status = %d, want 204", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/sessions/active", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("public sessions status = %d, want 204", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("user/me status = %d, want 401", recorder.Code)
	}
}

func TestAuthenticateAPIRequiresAPIKeyForSignedLink(t *testing.T) {
	keys := apiKeyLookupFunc(func(_ *http.Request, key string) (auth.Claims, bool, error) {
		return auth.Claims{}, key == "good", nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{PbUserID: "cookie-user"}, true, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || !claims.APIKeyPresent {
			t.Fatalf("signed-link must authenticate via API key: %#v ok=%v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without API key", recorder.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", nil)
	req.Header.Set(APIKeyHeader, "good")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPIAcceptsSessionCookieOnOAuthCallback(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key lookup should not run on OAuth callback")
		return auth.Claims{}, false, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{PbUserID: "user-1"}, true, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || claims.PbUserID != "user-1" {
			t.Fatalf("claims = %#v ok=%v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oauth/callback?code=x&state=y", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPIOAuthCallbackPassesThroughWithoutSession(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key lookup should not run on OAuth callback")
		return auth.Claims{}, false, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{}, false, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.ClaimsFromContext(r.Context()); ok {
			t.Fatal("expected no claims without session cookie")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/oauth/callback?code=x&state=y", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPIAcceptsSessionCookieOnUserMePath(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key lookup should not run when session is valid")
		return auth.Claims{}, false, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{PbUserID: "user-1"}, true, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || claims.PbUserID != "user-1" {
			t.Fatalf("claims = %#v ok=%v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPIAcceptsSessionCookieOnVAPath(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key lookup should not run when session is valid")
		return auth.Claims{}, false, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{PbUserID: "user-1", Role: "administrator"}, true, nil
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok || claims.PbUserID != "user-1" {
			t.Fatalf("claims = %#v ok=%v", claims, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/va", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestAuthenticateAPISessionLookupError(t *testing.T) {
	keys := apiKeyLookupFunc(func(*http.Request, string) (auth.Claims, bool, error) {
		t.Fatal("API key lookup should not run on session lookup error")
		return auth.Claims{}, false, nil
	})
	sessions := sessionLookupFunc(func(*http.Request) (auth.Claims, bool, error) {
		return auth.Claims{}, false, errors.New("redis down")
	})
	handler := AuthenticateAPI(keys, sessions)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not run")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

type apiKeyLookupFunc func(*http.Request, string) (auth.Claims, bool, error)

func (f apiKeyLookupFunc) Lookup(r *http.Request, apiKey string) (auth.Claims, bool, error) {
	return f(r, apiKey)
}

func TestRequireDiscordBotContext(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequireDiscordBotContext()(next)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(DiscordUserIDHeader, "user")
	req.Header.Set(DiscordServerIDHeader, "server")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}
