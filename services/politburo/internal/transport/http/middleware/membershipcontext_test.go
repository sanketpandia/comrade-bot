package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
)

type membershipResolverStub struct {
	vaID      string
	role      string
	pbUserID  string
	ok        bool
}

func (s membershipResolverStub) ResolveMembership(context.Context, string, string) (string, string, string, bool, error) {
	return s.vaID, s.role, s.pbUserID, s.ok, nil
}

func TestEnrichDiscordMembershipSetsPbUserIDWhenMissing(t *testing.T) {
	var got auth.Claims
	handler := EnrichDiscordMembership(membershipResolverStub{
		vaID: "va-1", role: "proletariat", pbUserID: "user-42", ok: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("missing claims")
		}
		got = claims
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{
		DsUserID:   "discord-1",
		DsServerID: "guild-1",
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got.PbUserID != "user-42" || got.PbServerID != "va-1" || got.Role != "proletariat" {
		t.Fatalf("claims = %+v", got)
	}
}

func TestEnrichDiscordMembershipPreservesSessionPbUserID(t *testing.T) {
	var got auth.Claims
	handler := EnrichDiscordMembership(membershipResolverStub{
		vaID: "va-1", role: "administrator", pbUserID: "resolved-user", ok: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := auth.ClaimsFromContext(r.Context())
		got = claims
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{
		PbUserID:   "session-user",
		DsUserID:   "discord-1",
		DsServerID: "guild-1",
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got.PbUserID != "session-user" {
		t.Fatalf("PbUserID = %q want session-user", got.PbUserID)
	}
}
