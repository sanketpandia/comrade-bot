package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/access/session"
	"infinite-experiment/politburo/internal/cache"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	gamesessions 	"infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/operations/contact"
	"infinite-experiment/politburo/internal/routes"
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
	return NewHandler(renderer, sessions, tickets, nil, nil, nil, nil, nil, nil, nil, false, nil, nil)
}

func TestLoginConsumesTicketThenSetsCookie(t *testing.T) {
	tickets := &ticketConsumerStub{ticket: &auth.LoginTicket{
		UserID: "u1", DiscordUserID: "ds-1", RedirectTo: routes.MapsFlightsActive,
	}}
	sessions := &sessionManagerStub{created: session.Session{SessionID: "sess-abc"}}
	handler := newUIHandler(t, sessions, tickets)

	req := httptest.NewRequest(http.MethodGet, routes.AuthLogin+"?token=once", nil)
	recorder := httptest.NewRecorder()
	handler.Login(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != routes.MapsFlightsActive {
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
	req := httptest.NewRequest(http.MethodGet, routes.AuthLogin+"?token=used", nil)
	recorder := httptest.NewRecorder()
	handler.Login(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if cookie := recorder.Header().Get("Set-Cookie"); strings.Contains(cookie, "session_id=sess") {
		t.Fatalf("expired login must not set session cookie: %q", cookie)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Sign-in link expired") || !strings.Contains(body, routes.Dashboard) {
		t.Fatalf("missing unauthorized copy: %s", body)
	}
	if !strings.Contains(body, `href="/"`) {
		t.Fatalf("missing home link: %s", body)
	}
}

func TestHomeRendersPublicLinks(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, routes.Home, nil)
	recorder := httptest.NewRecorder()
	handler.Home(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`href="` + routes.Dashboard + `"`,
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
			path:    routes.TermsConditions,
			handler: handler.TermsConditions,
			title:   "Terms of Service",
			marker:  "TERMS OF SERVICE",
		},
		{
			name:    "privacy",
			path:    routes.Privacy,
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
	return NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, submitter, nil, nil, nil, nil, nil, false, nil, nil)
}

func TestContactRendersForm(t *testing.T) {
	handler := newContactUIHandler(t, &contactSubmitterStub{})
	recorder := httptest.NewRecorder()
	handler.Contact(recorder, httptest.NewRequest(http.MethodGet, routes.Contact, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"<title>Contact</title>",
		`name="if_community_id"`,
		`name="message"`,
		`action="` + routes.Contact + `"`,
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
	handler.Contact(recorder, httptest.NewRequest(http.MethodGet, routes.Contact+"?sent=1", nil))
	body := recorder.Body.String()
	if !strings.Contains(body, "Thanks — your message was received.") {
		t.Fatalf("missing success copy: %s", body)
	}
}

func TestSubmitContactValidRedirects(t *testing.T) {
	submitter := &contactSubmitterStub{}
	handler := newContactUIHandler(t, submitter)
	req := httptest.NewRequest(http.MethodPost, routes.Contact, strings.NewReader("if_community_id=PilotOne&message=Hello+there"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.SubmitContact(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != routes.Contact+"?sent=1" {
		t.Fatalf("location = %q", loc)
	}
	if submitter.calls != 1 || submitter.ifc != "PilotOne" || submitter.msg != "Hello there" {
		t.Fatalf("create = %+v", submitter)
	}
}

func TestSubmitContactEmptyFieldsShowsError(t *testing.T) {
	submitter := &contactSubmitterStub{}
	handler := newContactUIHandler(t, submitter)
	req := httptest.NewRequest(http.MethodPost, routes.Contact, strings.NewReader("if_community_id=&message="))
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
	if !strings.Contains(body, `href="`+routes.Dashboard+`"`) || !strings.Contains(body, "https://github.com/sanketpandia/comrade-bot/issues/new") {
		t.Fatalf("missing not-found actions: %s", body)
	}
}

func TestNotFoundMachinePathsStayJSON(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	for _, path := range []string{routes.APIV1Prefix + "missing", routes.HealthPrefix + "nope", routes.MetricsPrefix + "extra"} {
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
	req := httptest.NewRequest(http.MethodGet, routes.StaticBrandPrefix+"logo.png", nil)
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
	req := httptest.NewRequest(http.MethodGet, routes.Dashboard, nil)
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`aria-label="Profile"`,
		`href="` + routes.Dashboard + `"`,
		`aria-current="page"`,
		"Live flights",
		`class="panel panel--glass panel--spaced stack"`,
		`class="card card--link"`,
		` flights</span>`,
		` pilots</span>`,
		`class="count-primary">—</span>`,
		`href="` + routes.MapsFlightsActive + `?serverId=expert"`,
		`href="` + routes.MapsFlightsActive + `?serverId=casual"`,
		`href="` + routes.MapsFlightsActive + `?serverId=training"`,
		routes.StaticPrefix + "css/tokens.css",
		routes.StaticPrefix + "css/base.css",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Index(body, `href="`+routes.Dashboard+`"`) > strings.Index(body, `aria-label="Profile"`) {
		t.Fatalf("nav order = profile then dashboard, want dashboard then profile")
	}
	if strings.Contains(body, `href="`+routes.AdminGeneral+`"`) {
		t.Fatalf("admin link should not show without administrator role")
	}
}

type sessionListStub struct {
	result gamesessions.ListActiveResult
	err    error
}

func (s sessionListStub) ListActive(context.Context) (gamesessions.ListActiveResult, error) {
	return s.result, s.err
}

type jsonCache struct {
	values map[string][]byte
}

func (c jsonCache) GetJSON(_ context.Context, key string, destination any) error {
	raw, ok := c.values[key]
	if !ok {
		return cache.ErrMiss
	}
	return json.Unmarshal(raw, destination)
}

func (c jsonCache) SetJSON(context.Context, string, any, time.Duration) error {
	return nil
}

func TestDashboardShowsServerCounts(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	raw, err := json.Marshal(domainflights.Snapshot{
		Tracks:     map[string]domainflights.FlightMotion{"a": {}, "b": {}},
		LastCached: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil,
		sessionListStub{result: gamesessions.ListActiveResult{Sessions: []gamesessions.ActiveSession{
			{NormalizedName: "expert", UserCount: 41},
			{NormalizedName: "training", UserCount: 0},
		}}},
		domainflights.NewReader(jsonCache{values: map[string][]byte{
			cache.KeyActiveFlights("expert"): raw,
		}}),
		false,
		nil,
		nil,
	)
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, httptest.NewRequest(http.MethodGet, routes.Dashboard, nil))
	body := recorder.Body.String()
	for _, want := range []string{
		`serverId=expert"`,
		`<span class="count-primary">2</span> flights`,
		`<span class="count-primary">41</span> pilots`,
		`<span class="count-primary">0</span> pilots`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Count(body, `<span class="count-primary">—</span>`) != 3 {
		t.Fatalf("missing counts = %d, want 3 dashes (casual flights and pilots, training flights)\n%s", strings.Count(body, `<span class="count-primary">—</span>`), body)
	}
}

func TestDashboardShowsAdminNavForAdministrator(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.Dashboard, nil), auth.Claims{Role: "administrator"})
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `href="`+routes.AdminGeneral+`"`) {
		t.Fatalf("missing admin nav link: %s", body)
	}
	if strings.Index(body, `href="`+routes.Dashboard+`"`) > strings.Index(body, `href="`+routes.AdminGeneral+`"`) {
		t.Fatalf("nav order = admin before dashboard, want dashboard then admin")
	}
}

func TestAdminRedirect(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	recorder := httptest.NewRecorder()
	handler.AdminRedirect(recorder, httptest.NewRequest(http.MethodGet, routes.Admin, nil))
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != routes.AdminGeneral {
		t.Fatalf("location = %q, want /admin/general", loc)
	}
}

func TestAdminRendersShell(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.AdminGeneral, nil), auth.Claims{Role: "administrator"})
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
		`>Live Mode</a>`,
		`class="section__title"`,
		`id="va-code"`,
		`readonly`,
		`/static/js/admin/general.mjs`,
		`app-main--wide`,
		`href="` + routes.AdminGeneral + `"`,
		`href="` + routes.AdminLive + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if !strings.Contains(body, `href="`+routes.AdminGeneral+`" aria-current="page"`) {
		t.Fatalf("general tab should be current page: %s", body)
	}
	if strings.Contains(body, `href="`+routes.AdminLive+`" aria-current="page"`) {
		t.Fatalf("live tab should not be current on general: %s", body)
	}
	if strings.Contains(body, "Link your in-game organisations") {
		t.Fatalf("live mode action should not render on general: %s", body)
	}
}

func TestAdminLiveShowsOrganisationLink(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, true, nil, nil)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.AdminLive+"?oauth=connected", nil), auth.Claims{Role: "administrator"})
	recorder := httptest.NewRecorder()
	handler.Admin(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`href="` + routes.AdminLive + `" aria-current="page"`,
		`id="admin-live-heading"`,
		`Each in-game organisation can be linked to one Discord server only`,
		`Link your in-game organisations`,
		`id="if-oauth-connect"`,
		`class="btn btn--primary"`,
		` disabled aria-disabled="true"`,
		`badge badge--success`,
		`/static/js/admin/oauth.mjs`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, `/static/js/admin/general.mjs`) {
		t.Fatalf("general script should not load on live mode: %s", body)
	}
	if strings.Contains(body, `id="va-code"`) {
		t.Fatalf("general fields should not render on live mode: %s", body)
	}
}

type vaLiveGrantStub struct {
	grant *valive.Grant
	err   error
}

func (s vaLiveGrantStub) GetByVAID(context.Context, string) (*valive.Grant, error) {
	return s.grant, s.err
}

func TestAdminLiveShowsConnectedFromActiveGrantWithoutQuery(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, true,
		vaLiveGrantStub{grant: &valive.Grant{Status: valive.GrantStatusActive}}, nil,
	)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.AdminLive, nil), auth.Claims{
		Role:       "administrator",
		PbServerID: "va-1",
	})
	recorder := httptest.NewRecorder()
	handler.Admin(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `badge badge--success`) {
		t.Fatalf("expected connected badge from grant: %s", body)
	}
	if !strings.Contains(body, `id="if-oauth-connect"`) || !strings.Contains(body, ` disabled aria-disabled="true"`) {
		t.Fatalf("connect button should be disabled when grant is active: %s", body)
	}
}

func TestAdminLiveOmitsLinkWhenOAuthDisabled(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.AdminLive, nil), auth.Claims{Role: "administrator"})
	recorder := httptest.NewRecorder()
	handler.Admin(recorder, req)
	body := recorder.Body.String()
	if strings.Contains(body, "Link your in-game organisations") || strings.Contains(body, `id="if-oauth-connect"`) {
		t.Fatal("organisation link should be hidden when oauth is not configured")
	}
	if !strings.Contains(body, "not configured") {
		t.Fatalf("missing unconfigured message: %s", body)
	}
}

func TestAdminForbiddenForNonAdmin(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.AdminGeneral, nil), auth.Claims{Role: "proletariat"})
	recorder := httptest.NewRecorder()
	handler.Admin(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestLiveModeRendersAircraftTabAndOrgSelect(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, false, nil, nil)
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraft, nil), auth.Claims{Role: "administrator"})
	recorder := httptest.NewRecorder()
	handler.LiveMode(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`id="live-org-select"`,
		`class="select"`,
		`href="` + routes.LiveAircraft + `" aria-current="page"`,
		`id="live-aircraft-fleet"`,
		`id="live-aircraft-row-menu"`,
		`/static/js/live/organizations.mjs`,
		`Live Mode`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestLiveModeForbiddenForNonAdmin(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := withClaims(httptest.NewRequest(http.MethodGet, routes.LiveAircraft, nil), auth.Claims{Role: "proletariat"})
	recorder := httptest.NewRecorder()
	handler.LiveMode(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestActiveFlightsMapRenders(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, routes.MapsFlightsActive, nil)
	recorder := httptest.NewRecorder()
	handler.ActiveFlightsMap(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `id="leaflet-map"`) || !strings.Contains(body, `id="maps-filters"`) {
		t.Fatalf("missing map shell: %s", body)
	}
	if !strings.Contains(body, `id="maps-detail-panel"`) || !strings.Contains(body, `hidden`) {
		t.Fatalf("detail panel should start closed: %s", body)
	}
	for _, tab := range []string{">Flight<", ">Route<", ">User<"} {
		if !strings.Contains(body, tab) {
			t.Fatalf("missing tab %q: %s", tab, body)
		}
	}
	if !strings.Contains(body, `aria-label="Primary"`) || !strings.Contains(body, `maps-page`) {
		t.Fatalf("map is outside the app shell: %s", body)
	}
	if !strings.Contains(body, `hx-get="`+routes.MapsFlightsActiveFlights+`"`) {
		t.Fatalf("missing flights partial request: %s", body)
	}
	if !strings.Contains(body, `id="maps-server"`) {
		t.Fatalf("missing server select: %s", body)
	}
	if !strings.Contains(body, `hx-get="`+routes.MapsFlightsActiveServers) {
		t.Fatalf("server select must load options via HTMX: %s", body)
	}
	if !strings.Contains(body, `hx-target="this"`) {
		t.Fatalf("server select must swap into itself, not inherit form hx-target: %s", body)
	}
	if !strings.Contains(body, `id="maps-pilot-state"`) || !strings.Contains(body, `multiselect__trigger`) {
		t.Fatalf("pilot state should use the custom multi-select dropdown: %s", body)
	}
	if !strings.Contains(body, `id="maps-filters-apply"`) {
		t.Fatalf("missing apply filters control: %s", body)
	}
	if !strings.Contains(body, `id="maps-filters-toggle"`) || !strings.Contains(body, `maps-filters__icon-toggle`) {
		t.Fatalf("missing filters icon toggle: %s", body)
	}
	if !strings.Contains(body, `id="maps-filters-collapse"`) || !strings.Contains(body, `maps-filters__header`) {
		t.Fatalf("missing expanded filters header row: %s", body)
	}
	if strings.Contains(body, `keyup changed delay:300ms`) {
		t.Fatalf("filters should not search on keystroke: %s", body)
	}
}

func TestActiveFlightsMapEmbedHidesHeader(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, routes.MapsFlightsActive+"?mode=embed", nil)
	recorder := httptest.NewRecorder()
	handler.ActiveFlightsMap(recorder, req)
	body := recorder.Body.String()
	if strings.Contains(body, `aria-label="Primary"`) {
		t.Fatalf("embed mode should omit topbar: %s", body)
	}
	if !strings.Contains(body, `maps-shell--embed`) {
		t.Fatalf("missing embed shell class: %s", body)
	}
}

func TestMapServerOptionsFallback(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest(http.MethodGet, routes.MapsFlightsActiveServers+"?serverId=training", nil)
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
	req := httptest.NewRequest(http.MethodGet, routes.AuthLogout, nil)
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

func TestDashboardOmitsOrganisationLink(t *testing.T) {
	renderer, err := appui.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	handler := NewHandler(renderer, &sessionManagerStub{}, &ticketConsumerStub{}, nil, nil, nil, nil, nil, nil, nil, true, nil, nil)
	recorder := httptest.NewRecorder()
	handler.Dashboard(recorder, httptest.NewRequest(http.MethodGet, routes.Dashboard, nil))
	body := recorder.Body.String()
	if strings.Contains(body, "Link your in-game organisations") || strings.Contains(body, `id="if-oauth-connect"`) || strings.Contains(body, "/static/js/admin/oauth.mjs") {
		t.Fatalf("organisation link should live on admin, not the dashboard: %s", body)
	}
}
