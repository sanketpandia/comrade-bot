package ui

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/access/session"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	gamesessions 	"infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/valive"
	"infinite-experiment/politburo/internal/operations/contact"
	"infinite-experiment/politburo/internal/operations/operator"
	"infinite-experiment/politburo/internal/operations/reports"
	"infinite-experiment/politburo/internal/routes"
	"infinite-experiment/politburo/internal/transport/http/response"
	appui "infinite-experiment/politburo/internal/ui"
)

type ticketConsumer interface {
	Consume(ctx context.Context, token string) (*auth.LoginTicket, error)
}

type sessionManager interface {
	Create(ctx context.Context, input session.CreateInput) (session.Session, error)
	Delete(ctx context.Context, sessionID string) error
}

type contactSubmitter interface {
	Create(ctx context.Context, ifCommunityID, message string) (*contact.Submission, error)
}

type contactRateLimiter interface {
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

const (
	contactMaxIFCLen     = 30
	contactMaxMessageLen = 5000
	contactMaxBodyBytes  = 8192
)

type sessionLister interface {
	ListActive(ctx context.Context) (gamesessions.ListActiveResult, error)
}

type mapServer struct {
	ID      string
	Label   string
	Flights string
	Pilots  string
}

// liveMapServers are the three Infinite Flight servers the live map accepts as serverId.
func liveMapServers() []mapServer {
	return []mapServer{
		{ID: "expert", Label: "Expert"},
		{ID: "casual", Label: "Casual"},
		{ID: "training", Label: "Training"},
	}
}

func (h *Handler) dashboardServers(ctx context.Context) []mapServer {
	servers := liveMapServers()
	pilots := map[string]int{}
	if h.lister != nil {
		result, err := h.lister.ListActive(ctx)
		if err != nil {
			slog.Warn("list active sessions for dashboard", "error", err)
		} else {
			for _, session := range result.Sessions {
				id := strings.TrimSpace(session.NormalizedName)
				if id == "" {
					continue
				}
				pilots[id] = session.UserCount
			}
		}
	}
	for i, server := range servers {
		servers[i].Flights = countText(h.flightCount(ctx, server.ID))
		if count, ok := pilots[server.ID]; ok {
			servers[i].Pilots = strconv.Itoa(count)
		} else {
			servers[i].Pilots = "—"
		}
	}
	return servers
}

func (h *Handler) flightCount(ctx context.Context, serverID string) (int, bool) {
	if h.flights == nil {
		return 0, false
	}
	count, err := h.flights.CountActive(ctx, serverID)
	if err != nil {
		if !errors.Is(err, domainflights.ErrCacheMiss) && !errors.Is(err, domainflights.ErrCacheCorrupt) {
			slog.Warn("count active flights for dashboard", "server", serverID, "error", err)
		}
		return 0, false
	}
	return count, true
}

func countText(value int, ok bool) string {
	if !ok {
		return "—"
	}
	return strconv.Itoa(value)
}

func serverLabel(id string) string {
	parts := strings.Split(strings.TrimSpace(id), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

type vaLiveGrantReader interface {
	GetByVAID(ctx context.Context, vaID string) (*valive.Grant, error)
}

type Handler struct {
	renderer       *appui.Renderer
	sessions       sessionManager
	tickets        ticketConsumer
	lister         sessionLister
	flights        *domainflights.Reader
	reports        *reports.Repository
	contact        contactSubmitter
	contactLimiter contactRateLimiter
	operator       *operator.Service
	operators      []string
	ifOAuthEnabled bool
	vaLiveGrants   vaLiveGrantReader
}

func NewHandler(
	renderer *appui.Renderer,
	sessions sessionManager,
	tickets ticketConsumer,
	reportRepo *reports.Repository,
	contactRepo contactSubmitter,
	contactLimiter contactRateLimiter,
	operatorService *operator.Service,
	platformOperators []string,
	lister sessionLister,
	flightsReader *domainflights.Reader,
	ifOAuthEnabled bool,
	vaLiveGrants vaLiveGrantReader,
) *Handler {
	return &Handler{
		renderer: renderer, sessions: sessions, tickets: tickets, lister: lister,
		flights: flightsReader,
		reports: reportRepo, contact: contactRepo, contactLimiter: contactLimiter,
		operator: operatorService, operators: platformOperators,
		ifOAuthEnabled: ifOAuthEnabled,
		vaLiveGrants:   vaLiveGrants,
	}
}

type publicPage struct {
	Title         string
	Content       string
	Role          string
	Sent          bool
	FormError     string
	IFCommunityID string
	Message       string
}

// Home is the public landing page.
func (h *Handler) Home(w http.ResponseWriter, _ *http.Request) {
	h.renderPublic(w, http.StatusOK, publicPage{
		Title:   "Comrade Bot",
		Content: "home-content",
	})
}

// TermsConditions serves the public terms of service page.
func (h *Handler) TermsConditions(w http.ResponseWriter, _ *http.Request) {
	h.renderPublic(w, http.StatusOK, publicPage{
		Title:   "Terms of Service",
		Content: "terms-conditions-content",
	})
}

// PrivacyPolicy serves the public privacy policy page.
func (h *Handler) PrivacyPolicy(w http.ResponseWriter, _ *http.Request) {
	h.renderPublic(w, http.StatusOK, publicPage{
		Title:   "Privacy Policy",
		Content: "privacy-policy-content",
	})
}

// Contact shows the public contact form.
func (h *Handler) Contact(w http.ResponseWriter, r *http.Request) {
	h.renderPublic(w, http.StatusOK, publicPage{
		Title:   "Contact",
		Content: "contact-content",
		Sent:    strings.TrimSpace(r.URL.Query().Get("sent")) == "1",
	})
}

// SubmitContact accepts a contact form POST and stores the submission.
func (h *Handler) SubmitContact(w http.ResponseWriter, r *http.Request) {
	if h.contact == nil {
		http.Error(w, "contact unavailable", http.StatusServiceUnavailable)
		return
	}
	if limited, msg := h.contactRateLimited(r); limited {
		h.renderContactForm(w, publicPage{
			FormError: msg,
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, contactMaxBodyBytes)
	if err := r.ParseForm(); err != nil {
		h.renderContactForm(w, publicPage{
			FormError: "Message is too long or the form could not be read.",
		})
		return
	}

	ifc := strings.TrimSpace(r.FormValue("if_community_id"))
	message := strings.TrimSpace(r.FormValue("message"))
	if errMsg := validateContactForm(ifc, message); errMsg != "" {
		h.renderContactForm(w, publicPage{
			FormError:     errMsg,
			IFCommunityID: ifc,
			Message:       message,
		})
		return
	}

	if _, err := h.contact.Create(r.Context(), ifc, message); err != nil {
		slog.Error("create contact submission", "error", err)
		h.renderContactForm(w, publicPage{
			FormError:     "Could not save your message. Try again later.",
			IFCommunityID: ifc,
			Message:       message,
		})
		return
	}
	http.Redirect(w, r, routes.Contact+"?sent=1", http.StatusSeeOther)
}

func (h *Handler) renderContactForm(w http.ResponseWriter, page publicPage) {
	page.Title = "Contact"
	page.Content = "contact-content"
	h.renderPublic(w, http.StatusOK, page)
}

func validateContactForm(ifCommunityID, message string) string {
	if ifCommunityID == "" {
		return "Enter your Infinite Flight Community username."
	}
	if len(ifCommunityID) > contactMaxIFCLen {
		return fmt.Sprintf("IF Community username must be at most %d characters.", contactMaxIFCLen)
	}
	if message == "" {
		return "Enter a message."
	}
	if len(message) > contactMaxMessageLen {
		return fmt.Sprintf("Message must be at most %d characters.", contactMaxMessageLen)
	}
	return ""
}

func (h *Handler) contactRateLimited(r *http.Request) (bool, string) {
	if h.contactLimiter == nil {
		return false, ""
	}
	const limit = 10
	now := time.Now()
	minuteBucket := now.Unix() / 60
	key := fmt.Sprintf("ratelimit:submit:%s:%d", contactCallerID(r), minuteBucket)
	count, err := h.contactLimiter.Incr(r.Context(), key, 2*time.Minute)
	if err != nil {
		return false, ""
	}
	if int(count) > limit {
		retryAfter := (minuteBucket+1)*60 - now.Unix()
		if retryAfter < 1 {
			retryAfter = 1
		}
		return true, fmt.Sprintf("Too many submissions. Wait %d seconds and try again.", retryAfter)
	}
	return false, ""
}

func contactCallerID(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

// NotFound renders the public 404 page. Machine paths stay JSON.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	if machineNotFound(r.URL.Path) {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "not found")
		return
	}
	h.renderPublic(w, http.StatusNotFound, publicPage{
		Title:   "Not found",
		Content: "not-found-content",
	})
}

func machineNotFound(path string) bool {
	return strings.HasPrefix(path, routes.APIPrefix) ||
		path == routes.Health || strings.HasPrefix(path, routes.HealthPrefix) ||
		path == routes.Metrics || strings.HasPrefix(path, routes.MetricsPrefix)
}

func (h *Handler) renderPublic(w http.ResponseWriter, status int, page publicPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.renderer.Render(w, "public", page); err != nil {
		slog.Error("render public page", "template", page.Content, "error", err)
	}
}

func basePageData(r *http.Request, title, content string, extra map[string]any) map[string]any {
	claims, _ := auth.ClaimsFromContext(r.Context())
	data := map[string]any{
		"Title":   title,
		"Content": content,
		"Role":    claims.Role,
	}
	for key, value := range extra {
		data[key] = value
	}
	return data
}

// Dashboard is a stub landing page for the rewrite UI surface.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFromContext(r.Context())
	data := basePageData(r, "Dashboard", "dashboard-content", map[string]any{
		"Username": claims.DsUserID,
		"Servers":  h.dashboardServers(r.Context()),
	})
	if err := h.renderer.Render(w, "base", data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

// AdminRedirect sends /admin to the default General tab.
func (h *Handler) AdminRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, routes.AdminGeneral, http.StatusSeeOther)
}

func adminTab(path string) string {
	switch strings.TrimSuffix(path, "/") {
	case routes.AdminLive:
		return "live"
	default:
		return "general"
	}
}

// Admin is the VA administrator settings area.
func (h *Handler) Admin(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !claims.IsAdmin() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tab := adminTab(r.URL.Path)
	oauthQuery := strings.TrimSpace(r.URL.Query().Get("oauth"))
	liveGrantStatus := ""
	if tab == "live" && h.vaLiveGrants != nil {
		vaID := strings.TrimSpace(claims.PbServerID)
		if vaID != "" {
			grant, err := h.vaLiveGrants.GetByVAID(r.Context(), vaID)
			if err != nil {
				slog.Warn("load va live grant for admin ui", "va_id", vaID, "error", err)
			} else if grant != nil {
				liveGrantStatus = grant.Status
			}
		}
	}
	data := basePageData(r, "Admin", "admin-content", map[string]any{
		"AdminTab":          tab,
		"IFOAuthEnabled":    h.ifOAuthEnabled,
		"OAuthStatus":       oauthQuery,
		"LiveGrantStatus":   liveGrantStatus,
	})
	if err := h.renderer.Render(w, "base", data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

// LiveRedirect sends /live to the default Aircrafts tab.
func (h *Handler) LiveRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, routes.LiveAircraft, http.StatusSeeOther)
}

func liveTab(path string) string {
	switch strings.TrimSuffix(path, "/") {
	case routes.LiveFlights:
		return "flights"
	case routes.LiveSchedules:
		return "schedules"
	case routes.LiveAircraft:
		return "aircraft"
	default:
		return "aircraft"
	}
}

// LiveMode is the Infinite Flight Live dashboard for VA staff (bourgeoisie and administrator).
func (h *Handler) LiveMode(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !claims.IsStaff() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	data := basePageData(r, "Live Mode", "live-content", map[string]any{
		"LiveTab": liveTab(r.URL.Path),
	})
	if err := h.renderer.Render(w, "base", data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

// ActiveFlightsMap is the full-screen live flights map.
func (h *Handler) ActiveFlightsMap(w http.ResponseWriter, r *http.Request) {
	view := mapsPageViewFromRequest(r)
	data := basePageData(r, "Active flights", "maps-active-flights-content", map[string]any{
		"Maps": view,
	})
	if err := h.renderer.Render(w, "base", data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

// MapServerOptions returns the live-map server <option> list.
// HTMX swaps this into the map filter; flight positions stay in the map script
// because Leaflet consumes JSON, not HTML.
func (h *Handler) MapServerOptions(w http.ResponseWriter, r *http.Request) {
	selected := strings.TrimSpace(r.URL.Query().Get("serverId"))
	servers := liveMapServers()
	if h.lister != nil {
		result, err := h.lister.ListActive(r.Context())
		if err != nil {
			slog.Warn("list active sessions for map servers", "error", err)
		} else if listed := mapServersFromSessions(result.Sessions); len(listed) > 0 {
			servers = listed
		}
	}
	if err := h.renderer.Render(w, "map-server-options", map[string]any{
		"Servers":  servers,
		"Selected": selected,
	}); err != nil {
		slog.Error("render map server options", "error", err)
	}
}

func mapServersFromSessions(sessions []gamesessions.ActiveSession) []mapServer {
	servers := make([]mapServer, 0, len(sessions))
	seen := map[string]struct{}{}
	for _, session := range sessions {
		id := strings.TrimSpace(session.NormalizedName)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		servers = append(servers, mapServer{ID: id, Label: serverLabel(id)})
	}
	return servers
}

func (h *Handler) OperatorReports(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !slices.Contains(h.operators, claims.DsUserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	openReports, err := h.reports.ListOpen(r.Context())
	if err != nil {
		http.Error(w, "failed to load reports", http.StatusInternalServerError)
		return
	}
	if err := h.renderer.Render(w, "operator-reports", map[string]any{
		"Title":   "Operator reports",
		"Reports": openReports,
	}); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

func (h *Handler) ResolveOperatorReport(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !slices.Contains(h.operators, claims.DsUserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	reportID := chi.URLParam(r, "reportID")
	err := reports.ResolveOpenReport(r.Context(), h.reports, h.operator, reportID, claims.DsUserID)
	if err != nil {
		switch {
		case errors.Is(err, reports.ErrReportNotFound):
			http.Error(w, "report not found", http.StatusNotFound)
		case errors.Is(err, reports.ErrUnsupportedReportKind):
			http.Error(w, "unsupported report", http.StatusBadRequest)
		default:
			http.Error(w, "resolve failed", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, routes.OperatorReports, http.StatusSeeOther)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderExpired(w)
		return
	}
	ticket, err := h.tickets.Consume(r.Context(), token)
	if err != nil {
		if !errors.Is(err, auth.ErrInvalidTicket) {
			slog.Error("consume login ticket", "error", err)
		}
		h.renderExpired(w)
		return
	}
	created, err := h.sessions.Create(r.Context(), session.CreateInput{
		UserID:          ticket.UserID,
		DiscordID:       ticket.DiscordUserID,
		DiscordServerID: ticket.DiscordServerID,
		VaID:            ticket.VaID,
		Role:            ticket.Role,
		Username:        ticket.Username,
	})
	if err != nil {
		slog.Error("create session from login ticket", "error", err)
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}
	session.SetCookie(w, r, created.SessionID)
	http.Redirect(w, r, ticket.RedirectTo, http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Delete(r.Context(), session.SessionIDFromRequest(r)); err != nil {
		slog.Error("delete session on logout", "error", err)
	}
	session.ClearCookie(w, r)
	http.Redirect(w, r, auth.DefaultRedirect, http.StatusSeeOther)
}

func (h *Handler) renderExpired(w http.ResponseWriter) {
	h.renderPublic(w, http.StatusUnauthorized, publicPage{
		Title:   "Unauthorized",
		Content: "unauthorized-content",
	})
}

// Brand serves logo, favicon, and the loading clip under /static/brand/.
// The shared prefix is what a CDN path rule can cache.
func Brand() http.Handler {
	sub, err := fs.Sub(appui.Assets, "static/brand")
	if err != nil {
		panic("ui brand assets missing: " + err.Error())
	}
	files := http.StripPrefix(routes.StaticBrandPrefix, http.FileServer(http.FS(sub)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800")
		files.ServeHTTP(w, r)
	})
}

// Static serves embedded CSS/JS under /static/.
func Static() http.Handler {
	sub, err := fs.Sub(appui.Assets, "static")
	if err != nil {
		panic("ui static assets missing: " + err.Error())
	}
	return http.StripPrefix(routes.StaticPrefix, http.FileServer(http.FS(sub)))
}
