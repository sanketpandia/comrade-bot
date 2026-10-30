package ui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/access/session"
	"infinite-experiment/politburo/internal/operations/contact"
	appui "infinite-experiment/politburo/internal/ui"
)

type ticketConsumerStub struct {
	ticket *auth.LoginTicket
	err    error
	calls  int
}

func (s *ticketConsumerStub) Consume(context.Context, string) (*auth.LoginTicket, error) {
	s.calls++
	return s.ticket, s.err
}

type sessionManagerStub struct {
	created session.Session
	create  error
	deleted string
}

func (s *sessionManagerStub) Create(context.Context, session.CreateInput) (session.Session, error) {
	return s.created, s.create
}

func (s *sessionManagerStub) Delete(_ context.Context, sessionID string) error {
	s.deleted = sessionID
	return nil
}

func withClaims(r *http.Request, claims auth.Claims) *http.Request {
	return r.WithContext(auth.SetClaims(r.Context(), claims))
}

func newUIHandler(t *testing.T, sessions sessionManager, tickets ticketConsumer) *Handler {
	t.Helper()
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	return NewHandler(renderer, sessions, tickets, nil, nil, nil, nil, nil, nil, false)
}

func TestLoginConsumesTicketThenSetsCookie(t *testing.T) {
	tickets := &ticketConsumerStub{ticket: &auth.LoginTicket{
		UserID: "u1", DiscordUserID: "ds-1", RedirectTo: "/maps/flights/active",
	}}
	sessions := &sessionManagerStub{created: session.Session{SessionID: "sess-abc"}}
	handler := newUIHandler(t, sessions, tickets)

	req := httptest.NewRequest(http.MethodGet, "/auth/login?token=once", nil)
	recorder := httptest.NewRecorder()
	handler.Login(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != "/maps/flights/active" {
		t.Fatalf("location = %q", loc)
	}
	cookie := recorder.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "session_id=sess-abc") {
		t.Fatalf("set-cookie = %q", cookie)
	}
	if tickets.calls != 1 {
		t.Fatalf("consume calls = %d", tickets.calls)
	}
}

func TestLoginReplayRendersExpired(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{err: auth.ErrInvalidTicket})
	req := httptest.NewRequest(http.MethodGet, "/auth/login?token=used", nil)
	recorder := httptest.NewRecorder()
	handler.Login(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if cookie := recorder.Header().Get("Set-Cookie"); strings.Contains(cookie, "session_id=sess") {
		t.Fatalf("expired login must not set session cookie: %q", cookie)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Sign-in link expired") || !strings.Contains(body, "/dashboard") {
		t.Fatalf("missing unauthorized copy: %s", body)
	}
	if !strings.Contains(body, `href="/"`) {
		t.Fatalf("missing home link: %s", body)
	}
}

func TestHomeRendersPublicLinks(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.Home(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`href="/dashboard"`,
		`href="https://sanketpandia.github.io/comrade-bot/"`,
		`href="https://github.com/sanketpandia/comrade-bot"`,
		`aria-label="Profile"`,
		`/static/brand/favicon.png`,
		`/static/brand/loading.mp4`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestLegalPagesRenderPublicShell(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	tests := []struct {
		name    string
		path    string
		handler http.HandlerFunc
		title   string
		marker  string
	}{
		{
			name:    "terms",
			path:    "/terms-conditions",
			handler: handler.TermsConditions,
			title:   "Terms of Service",
			marker:  "TERMS OF SERVICE",
		},
		{
			name:    "privacy",
			path:    "/privacy-policy",
			handler: handler.PrivacyPolicy,
			title:   "Privacy Policy",
			marker:  "PRIVACY POLICY",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			recorder := httptest.NewRecorder()
			tc.handler(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", recorder.Code)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, "<title>"+tc.title+"</title>") {
				t.Fatalf("missing title %q", tc.title)
			}
			if !strings.Contains(body, tc.marker) {
				t.Fatalf("missing marker %q", tc.marker)
			}
			if !strings.Contains(body, `class="public-footer"`) {
				t.Fatalf("missing public footer")
			}
		})
	}
}

type contactSubmitterStub struct {
	calls int
	ifc   string
	msg   string
	err   error
}

func (s *contactSubmitterStub) Create(_ context.Context, ifCommunityID, message string) (*contact.Submission, error) {
	s.calls++
	s.ifc = ifCommunityID
	s.msg = message
	if s.err != nil {
		return nil, s.err
	}
	return &contact.Submission{ID: "sub-1", IFCommunityID: ifCommunityID, Message: message}, nil
}

func newContactUIHandler(t *testing.T, submitter contactSubmitter) *Handler {
	t.Helper()
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	return NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, submitter, nil, nil, nil, nil, false)
}

func TestContactRendersForm(t *testing.T) {
	handler := newContactUIHandler(t, &contactSubmitterStub{})
	recorder := httptest.NewRecorder()
	handler.Contact(recorder, httptest.NewRequest(http.MethodGet, "/contact", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"<title>Contact</title>",
		`name="if_community_id"`,
		`name="message"`,
		`action="/contact"`,
		`class="public-footer"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body", want)
		}
	}
}

func TestContactSentQueryShowsSuccess(t *testing.T) {
	handler := newContactUIHandler(t, &contactSubmitterStub{})
	recorder := httptest.NewRecorder()
	handler.Contact(recorder, httptest.NewRequest(http.MethodGet, "/contact?sent=1", nil))
	body := recorder.Body.String()
	if !strings.Contains(body, "Thanks — your message was received.") {
		t.Fatalf("missing success copy: %s", body)
	}
}

func TestSubmitContactValidRedirects(t *testing.T) {
	submitter := &contactSubmitterStub{}
	handler := newContactUIHandler(t, submitter)
	req := httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader("if_community_id=PilotOne&message=Hello+there"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.SubmitContact(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != "/contact?sent=1" {
		t.Fatalf("location = %q", loc)
	}
	if submitter.calls != 1 || submitter.ifc != "PilotOne" || submitter.msg != "Hello there" {
		t.Fatalf("create = %+v", submitter)
	}
}

func TestSubmitContactEmptyFieldsShowsError(t *testing.T) {
	submitter := &contactSubmitterStub{}
	handler := newContactUIHandler(t, submitter)
	req := httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader("if_community_id=&message="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.SubmitContact(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Enter your Infinite Flight Community username.") {
		t.Fatalf("missing validation copy: %s", body)
	}
	if submitter.calls != 0 {
		t.Fatalf("create calls = %d, want 0", submitter.calls)
	}
}

func TestNotFoundRendersHTML(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, "/no-such-page", nil)
	recorder := httptest.NewRecorder()
	handler.NotFound(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Page not found") || !strings.Contains(body, "This page does not exist.") {
		t.Fatalf("missing not-found copy: %s", body)
	}
	if !strings.Contains(body, `href="/dashboard"`) || !strings.Contains(body, "https://github.com/sanketpandia/comrade-bot/issues/new") {
		t.Fatalf("missing not-found actions: %s", body)
	}
}

func TestNotFoundMachinePathsStayJSON(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	for _, path := range []string{"/api/v1/missing", "/health/nope", "/metrics/extra"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		handler.NotFound(recorder, req)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, recorder.Code)
		}
		if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("%s content-type = %q", path, ct)
		}
		if !strings.Contains(recorder.Body.String(), `"NOT_FOUND"`) {
			t.Fatalf("%s body = %s", path, recorder.Body.String())
		}
	}
}

func TestBrandCacheControl(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/brand/logo.png", nil)
	recorder := httptest.NewRecorder()
	Brand().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=604800" {
		t.Fatalf("cache-control = %q", got)
	}
	if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "image/png") {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestDashboardRendersShell(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`aria-label="Profile"`,
		`href="/dashboard"`,
		`aria-current="page"`,
		"Live flights",
		`href="/maps/flights/active?serverId=expert"`,
		`href="/maps/flights/active?serverId=casual"`,
		`href="/maps/flights/active?serverId=training"`,
		"/static/css/base.css",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Index(body, `href="/dashboard"`) > strings.Index(body, `aria-label="Profile"`) {
		t.Fatalf("nav order = profile then dashboard, want dashboard then profile")
	}
	if strings.Contains(body, `href="/admin/general"`) {
		t.Fatalf("admin link should not show without administrator role")
	}
}

func TestDashboardShowsAdminNavForAdministrator(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, "/dashboard", nil), auth.Claims{Role: "administrator"})
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `href="/admin/general"`) {
		t.Fatalf("missing admin nav link: %s", body)
	}
	if strings.Index(body, `href="/dashboard"`) > strings.Index(body, `href="/admin/general"`) {
		t.Fatalf("nav order = admin before dashboard, want dashboard then admin")
	}
}

func TestAdminRedirect(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	recorder := httptest.NewRecorder()
	handler.AdminRedirect(recorder, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != "/admin/general" {
		t.Fatalf("location = %q, want /admin/general", loc)
	}
}

func TestAdminRendersShell(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, "/admin/general", nil), auth.Claims{Role: "administrator"})
	recorder := httptest.NewRecorder()
	handler.Admin(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`<h1>Admin</h1>`,
		`aria-label="Admin sections"`,
		`>General</a>`,
		`id="va-code"`,
		`readonly`,
		`/static/js/admin/general.mjs`,
		`app-main--wide`,
		`href="/admin/general"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if !strings.Contains(body, `href="/admin/general" aria-current="page"`) {
		t.Fatalf("admin tab should be current page: %s", body)
	}
}

func TestAdminForbiddenForNonAdmin(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, "/admin/general", nil), auth.Claims{Role: "proletariat"})
	recorder := httptest.NewRecorder()
	handler.Admin(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestActiveFlightsMapRenders(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, "/maps/flights/active", nil)
	recorder := httptest.NewRecorder()
	handler.ActiveFlightsMap(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `id="leaflet-map"`) || !strings.Contains(body, `id="maps-filters"`) {
		t.Fatalf("missing map shell: %s", body)
	}
	if !strings.Contains(body, `id="maps-detail-panel"`) {
		t.Fatalf("missing flight detail panel: %s", body)
	}
	if !strings.Contains(body, `aria-label="Primary"`) || !strings.Contains(body, `class="maps-page"`) {
		t.Fatalf("map is outside the app shell: %s", body)
	}
	if !strings.Contains(body, "/static/js/maps/active-flights.mjs") || !strings.Contains(body, "/static/js/htmx.min.js") {
		t.Fatalf("missing page script: %s", body)
	}
	if !strings.Contains(body, `hx-get="/maps/flights/active/servers"`) {
		t.Fatalf("missing server options request: %s", body)
	}
}

func TestMapServerOptionsFallback(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, "/maps/flights/active/servers?serverId=training", nil)
	recorder := httptest.NewRecorder()
	handler.MapServerOptions(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`<option value="expert">Expert</option>`,
		`<option value="casual">Casual</option>`,
		`<option value="training" selected>Training</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestLogoutDeletesSessionAndCookie(t *testing.T) {
	sessions := &sessionManagerStub{}
	handler := newUIHandler(t, sessions, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "sess-abc"})
	recorder := httptest.NewRecorder()
	handler.Logout(recorder, req)
	if sessions.deleted != "sess-abc" {
		t.Fatalf("deleted = %q", sessions.deleted)
	}
	cookie := recorder.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "session_id=") || (!strings.Contains(cookie, "Max-Age=0") && !strings.Contains(cookie, "Max-Age=-1")) {
		t.Fatalf("set-cookie = %q", cookie)
	}
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
}

func TestDashboardShowsConnectWhenOAuthConfigured(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, true)
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	body := recorder.Body.String()
	if !strings.Contains(body, "Connect Infinite Flight") || !strings.Contains(body, `id="if-oauth-connect"`) {
		t.Fatalf("missing oauth tile: %s", body)
	}
}

func TestDashboardOmitsConnectWhenOAuthDisabled(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if strings.Contains(recorder.Body.String(), "Connect Infinite Flight") {
		t.Fatal("oauth tile should be hidden when client is not configured")
	}
}
