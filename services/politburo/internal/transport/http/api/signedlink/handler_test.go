package signedlink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/routes"
)

type userStub struct {
	user *users.User
	err  error
}

func (s userStub) GetByDiscordID(context.Context, string) (*users.User, error) {
	return s.user, s.err
}

type ticketStub struct {
	token string
	err   error
	got   auth.LoginTicket
}

func (s *ticketStub) Issue(_ context.Context, ticket auth.LoginTicket) (string, error) {
	s.got = ticket
	return s.token, s.err
}

func TestGenerateSignedLinkRequiresDiscordUser(t *testing.T) {
	handler := NewHandler(userStub{}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"DASHBOARD"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestGenerateSignedLinkUnauthorizedWithoutClaims(t *testing.T) {
	handler := NewHandler(userStub{}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", nil)
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestGenerateSignedLinkUserNotFound(t *testing.T) {
	handler := NewHandler(userStub{}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"DASHBOARD"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestGenerateSignedLinkMissingType(t *testing.T) {
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestGenerateSignedLinkInvalidType(t *testing.T) {
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"BOGUS"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestGenerateSignedLinkCustomMissingRedirect(t *testing.T) {
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"CUSTOM"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestGenerateSignedLinkRejectsAbsoluteRedirect(t *testing.T) {
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"CUSTOM","redirectTo":"https://evil.example"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestGenerateSignedLinkStoresOptionalServerAndDefaultRedirect(t *testing.T) {
	username := "pilot"
	tickets := &ticketStub{token: "tok"}
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1", Username: &username}}, tickets, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", bytes.NewBufferString(`{"type":"DASHBOARD"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{
		APIKeyPresent: true, DsUserID: "ds-1", DsServerID: "guild-1",
	}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if tickets.got.DiscordServerID != "guild-1" || tickets.got.RedirectTo != routes.Dashboard || tickets.got.Username != "pilot" {
		t.Fatalf("ticket = %#v", tickets.got)
	}
	var payload map[string]map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data := payload["data"]
	if data["url"] != "https://ui.example/auth/login?token=tok" || data["expiresIn"] != float64(600) || data["redirectTo"] != routes.Dashboard {
		t.Fatalf("data = %#v", data)
	}
}

func TestGenerateSignedLinkLiveMap(t *testing.T) {
	tickets := &ticketStub{token: "tok"}
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}}, tickets, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"LIVE_MAP"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if tickets.got.RedirectTo != routes.MapsFlightsActive {
		t.Fatalf("ticket = %#v", tickets.got)
	}
}

func TestGenerateSignedLinkCustomRedirect(t *testing.T) {
	tickets := &ticketStub{token: "tok"}
	handler := NewHandler(userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}}, tickets, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"CUSTOM","redirectTo":"/maps/flights/active"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if tickets.got.RedirectTo != routes.MapsFlightsActive {
		t.Fatalf("ticket = %#v", tickets.got)
	}
}

func TestGenerateSignedLinkLookupError(t *testing.T) {
	handler := NewHandler(userStub{err: errors.New("db down")}, &ticketStub{token: "tok"}, nil, nil, nil, "https://ui.example")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"DASHBOARD"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{APIKeyPresent: true, DsUserID: "ds-1"}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

type membershipStub struct {
	vaID, role string
	ok         bool
	err        error
}

func (s membershipStub) ResolveMembership(context.Context, string, string) (string, string, bool, error) {
	return s.vaID, s.role, s.ok, s.err
}

type vaLookupStub struct {
	configured bool
	err        error
}

func (s vaLookupStub) IsConfiguredVA(context.Context, string) (bool, error) {
	return s.configured, s.err
}

type mapLinkStub struct {
	path string
	err  error
}

func (s mapLinkStub) ActiveFlightsMapPath(context.Context, string) (string, error) {
	return s.path, s.err
}

func TestGenerateSignedLinkLiveMapWithVAFilters(t *testing.T) {
	filteredPath := routes.MapsFlightsActive + "?serverId=expert&vaCode=TST"
	tickets := &ticketStub{token: "tok"}
	handler := NewHandler(
		userStub{user: &users.User{ID: "u1", DiscordID: "ds-1"}},
		tickets,
		membershipStub{vaID: "va-1", role: "member", ok: true},
		vaLookupStub{configured: true},
		mapLinkStub{path: filteredPath},
		"https://ui.example",
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/signed-link", strings.NewReader(`{"type":"LIVE_MAP"}`))
	req = req.WithContext(auth.SetClaims(req.Context(), auth.Claims{
		APIKeyPresent: true, DsUserID: "ds-1", DsServerID: "guild-1",
	}))
	recorder := httptest.NewRecorder()
	handler.GenerateSignedLink(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if tickets.got.RedirectTo != filteredPath {
		t.Fatalf("ticket redirect = %q, want %q", tickets.got.RedirectTo, filteredPath)
	}
}
