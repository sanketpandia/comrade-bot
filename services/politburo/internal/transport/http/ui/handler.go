package ui

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/access/session"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/operations/operator"
	"infinite-experiment/politburo/internal/operations/reports"
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

type sessionLister interface {
	ListActive(ctx context.Context) (gamesessions.ListActiveResult, error)
}

type mapServer struct {
	ID    string
	Label string
}

// liveMapServers are the three Infinite Flight servers the live map accepts as serverId.
func liveMapServers() []mapServer {
	return []mapServer{
		{ID: "expert", Label: "Expert"},
		{ID: "casual", Label: "Casual"},
		{ID: "training", Label: "Training"},
	}
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

type Handler struct {
	renderer  *appui.Renderer
	sessions  sessionManager
	tickets   ticketConsumer
	lister    sessionLister
	reports   *reports.Repository
	operator  *operator.Service
	operators []string
}

func NewHandler(
	renderer *appui.Renderer,
	sessions sessionManager,
	tickets ticketConsumer,
	reportRepo *reports.Repository,
	operatorService *operator.Service,
	platformOperators []string,
	lister sessionLister,
) *Handler {
	return &Handler{
		renderer: renderer, sessions: sessions, tickets: tickets, lister: lister,
		reports: reportRepo, operator: operatorService, operators: platformOperators,
	}
}

type publicPage struct {
	Title   string
	Content string
	Role    string
}

// Home is the public landing page.
func (h *Handler) Home(w http.ResponseWriter, _ *http.Request) {
	h.renderPublic(w, http.StatusOK, publicPage{
		Title:   "Comrade Bot",
		Content: "home-content",
	})
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
	return strings.HasPrefix(path, "/api/") ||
		path == "/health" || strings.HasPrefix(path, "/health/") ||
		path == "/metrics" || strings.HasPrefix(path, "/metrics/")
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
		"Servers":  liveMapServers(),
	})
	if err := h.renderer.Render(w, "base", data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

// AdminRedirect sends /admin to the default General tab.
func (h *Handler) AdminRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/general", http.StatusSeeOther)
}

// Admin is the VA administrator settings area.
func (h *Handler) Admin(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || !claims.IsAdmin() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	data := basePageData(r, "Admin", "admin-content", map[string]any{
		"AdminTab": "general",
	})
	if err := h.renderer.Render(w, "base", data); err != nil {
		http.Error(w, "template render failed", http.StatusInternalServerError)
	}
}

// ActiveFlightsMap is the full-screen live flights map.
func (h *Handler) ActiveFlightsMap(w http.ResponseWriter, r *http.Request) {
	data := basePageData(r, "Active flights", "maps-active-flights-content", map[string]any{
		"ServerID": strings.TrimSpace(r.URL.Query().Get("serverId")),
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
	http.Redirect(w, r, "/operator/reports", http.StatusSeeOther)
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
	files := http.StripPrefix("/static/brand/", http.FileServer(http.FS(sub)))
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
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}
