package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/app"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"infinite-experiment/politburo/internal/transport/http/api/gameflights"
	"infinite-experiment/politburo/internal/transport/http/api/gamesessions"
	"infinite-experiment/politburo/internal/transport/http/api/health"
	"infinite-experiment/politburo/internal/transport/http/api/identity"
	"infinite-experiment/politburo/internal/transport/http/api/signedlink"
	userapi "infinite-experiment/politburo/internal/transport/http/api/user"
	vaapi "infinite-experiment/politburo/internal/transport/http/api/va"
	appmiddleware "infinite-experiment/politburo/internal/transport/http/middleware"
	"infinite-experiment/politburo/internal/transport/http/response"
	uihttp "infinite-experiment/politburo/internal/transport/http/ui"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	app *app.App
}

func NewServer(application *app.App) *Server {
	return &Server{app: application}
}

func (s *Server) Run(parent context.Context) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	router := s.router()
	server := &stdhttp.Server{
		Addr:         ":" + s.app.Config.HTTP.Port,
		Handler:      router,
		ReadTimeout:  s.app.Config.HTTP.ReadTimeout,
		WriteTimeout: s.app.Config.HTTP.WriteTimeout,
		IdleTimeout:  s.app.Config.HTTP.IdleTimeout,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	}
	if s.app.Config.Jobs.Enabled {
		s.app.Scheduler.Start()
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("http server started", "address", server.Addr)
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, stdhttp.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.app.Config.HTTP.ShutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func (s *Server) router() stdhttp.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(appmiddleware.CORS(s.app.Config.HTTP.AllowedOrigins))
	router.Use(middleware.Recoverer)
	router.Use(appmiddleware.AccessLog(s.app.Metrics))
	router.Use(appmiddleware.AuthenticateAPI(s.app.APIKeys, s.app.Sessions))

	healthHandler := health.NewHandler(s.app.DB, s.app.Cache, s.app.Config.Jobs.Enabled, s.app.StartedAt)
	sessionsHandler := gamesessions.NewHandler(s.app.SessionsReader)
	flightsHandler := gameflights.NewHandler(s.app.FlightsReader, s.app.Metrics)
	signedLinkHandler := signedlink.NewHandler(s.app.Users, s.app.Tickets, s.app.Resolver, s.app.VALookup, s.app.Config.Auth.UIBaseURL)
	identityHandler := identity.NewHandler(
		s.app.Registration, s.app.Membership, s.app.VAInit, s.app.Status,
		s.app.Reports, s.app.Operator, s.app.VirtualAirlines,
	)
	vaHandler := vaapi.NewHandler(s.app.VirtualAirlines)
	userHandler := userapi.NewHandler(s.app.Users)
	uiHandler := uihttp.NewHandler(s.app.UI, s.app.Sessions, s.app.Tickets, s.app.Reports, s.app.Operator, s.app.Config.PlatformOperatorDiscordIDs, s.app.SessionsReader)
	handler := apiHandler{
		health: healthHandler, sessions: sessionsHandler, flights: flightsHandler,
		signedLink: signedLinkHandler, identity: identityHandler, va: vaHandler, user: userHandler,
		registrationMiddleware: registrationRouteMiddleware(s.app),
		godMiddleware:          godRouteMiddleware(s.app),
		vaAdminMiddleware:      vaAdminRouteMiddleware(s.app),
		currentUserMiddleware:  currentUserRouteMiddleware(s.app),
	}

	// Public ops + OpenAPI machine API (paths include /health/* and /api/v1/*).
	// Game GETs accept a session cookie or API key; other /api/v1 paths need a key.
	politburoapi.HandlerWithOptions(handler, politburoapi.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: writeParameterError,
	})
	router.Handle("/metrics", promhttp.HandlerFor(s.app.Metrics.Prometheus, promhttp.HandlerOpts{}))

	router.Get("/auth/login", uiHandler.Login)
	router.Get("/auth/logout", uiHandler.Logout)
	router.Group(func(bot chi.Router) {
		bot.Use(
			appmiddleware.RequireDiscordBotContext(),
			appmiddleware.EnrichDiscordMembership(s.app.Resolver),
			appmiddleware.RateLimit(s.app.Cache, appmiddleware.RateLimitGroupRegistration, appmiddleware.RateLimitRegistration),
		)
		bot.Get("/api/v1/user/status", identityHandler.Status)
		bot.Post("/api/v1/memberships/join", identityHandler.Join)
		bot.Post("/api/v1/reports/occupied-ifc", identityHandler.ReportOccupiedIFC)
	})

	router.Group(func(op chi.Router) {
		op.Use(appmiddleware.RequirePlatformOperator(s.app.Config.PlatformOperatorDiscordIDs))
		op.Get("/api/v1/operator/reports", identityHandler.ListOperatorReports)
		op.Post("/api/v1/operator/reports/{reportID}/resolve", identityHandler.ResolveOperatorReport)
	})

	router.Group(func(ui chi.Router) {
		ui.Use(appmiddleware.UISessionAuth(s.app.Sessions))
		ui.Get("/dashboard", uiHandler.Dashboard)
		ui.Get("/dashboard/", uiHandler.Dashboard)
		ui.Get("/admin", uiHandler.AdminRedirect)
		ui.Get("/admin/", uiHandler.AdminRedirect)
		ui.Get("/admin/general", uiHandler.Admin)
		ui.Get("/admin/general/", uiHandler.Admin)
		ui.Get("/maps/flights/active", uiHandler.ActiveFlightsMap)
		ui.Get("/maps/flights/active/servers", uiHandler.MapServerOptions)
		ui.Get("/operator/reports", uiHandler.OperatorReports)
		ui.Post("/operator/reports/{reportID}/resolve", uiHandler.ResolveOperatorReport)
	})
	router.Get("/", uiHandler.Home)
	router.Handle("/static/brand/*", uihttp.Brand())
	router.Handle("/static/*", uihttp.Static())
	router.NotFound(uiHandler.NotFound)

	return router
}

type apiHandler struct {
	health                 *health.Handler
	sessions               *gamesessions.Handler
	flights                *gameflights.Handler
	signedLink             *signedlink.Handler
	identity               *identity.Handler
	va                     *vaapi.Handler
	user                   *userapi.Handler
	registrationMiddleware []func(stdhttp.Handler) stdhttp.Handler
	godMiddleware          []func(stdhttp.Handler) stdhttp.Handler
	vaAdminMiddleware      []func(stdhttp.Handler) stdhttp.Handler
	currentUserMiddleware  []func(stdhttp.Handler) stdhttp.Handler
}

func (h apiHandler) GetHealthStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.health.GetStatus(w, r)
}

func (h apiHandler) GetActiveSessions(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.sessions.GetActiveSessions(w, r)
}

func (h apiHandler) GetActiveFlights(w stdhttp.ResponseWriter, r *stdhttp.Request, normalizedServerName string, params politburoapi.GetActiveFlightsParams) {
	h.flights.GetActiveFlights(w, r, normalizedServerName, params)
}

func (h apiHandler) GetActiveFlight(w stdhttp.ResponseWriter, r *stdhttp.Request, flightID openapi_types.UUID) {
	h.flights.GetActiveFlight(w, r, flightID)
}

func (h apiHandler) GenerateSignedLink(w stdhttp.ResponseWriter, r *stdhttp.Request, _ politburoapi.GenerateSignedLinkParams) {
	h.signedLink.GenerateSignedLink(w, r)
}

func (h apiHandler) CreateUser(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.CreateUserParams) {
	h.withMiddleware(h.registrationMiddleware, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.identity.CreateUser(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) CreateServer(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.CreateServerParams) {
	h.withMiddleware(h.registrationMiddleware, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.identity.CreateServer(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) VerifyGodMode(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.VerifyGodModeParams) {
	h.withMiddleware(h.godMiddleware, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.identity.VerifyGod(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) BanUser(w stdhttp.ResponseWriter, r *stdhttp.Request, params politburoapi.BanUserParams) {
	h.withMiddleware(h.godMiddleware, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.identity.BanUser(w, r, params)
	})).ServeHTTP(w, r)
}

func (h apiHandler) GetVA(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.withMiddleware(h.vaAdminMiddleware, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.va.GetVA(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) GetCurrentUser(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	h.withMiddleware(h.currentUserMiddleware, stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.user.GetMe(w, r)
	})).ServeHTTP(w, r)
}

func (h apiHandler) withMiddleware(chain []func(stdhttp.Handler) stdhttp.Handler, handler stdhttp.Handler) stdhttp.Handler {
	for i := len(chain) - 1; i >= 0; i-- {
		handler = chain[i](handler)
	}
	return handler
}

func registrationRouteMiddleware(application *app.App) []func(stdhttp.Handler) stdhttp.Handler {
	return []func(stdhttp.Handler) stdhttp.Handler{
		appmiddleware.RequireDiscordBotContext(),
		appmiddleware.EnrichDiscordMembership(application.Resolver),
		appmiddleware.RateLimit(application.Cache, appmiddleware.RateLimitGroupRegistration, appmiddleware.RateLimitRegistration),
	}
}

func godRouteMiddleware(application *app.App) []func(stdhttp.Handler) stdhttp.Handler {
	return []func(stdhttp.Handler) stdhttp.Handler{
		appmiddleware.RequireDiscordBotContext(),
		appmiddleware.RequirePlatformOperator(application.Config.PlatformOperatorDiscordIDs),
	}
}

func vaAdminRouteMiddleware(application *app.App) []func(stdhttp.Handler) stdhttp.Handler {
	return []func(stdhttp.Handler) stdhttp.Handler{
		appmiddleware.EnrichDiscordMembership(application.Resolver),
		appmiddleware.RequireAdmin(),
	}
}

func currentUserRouteMiddleware(application *app.App) []func(stdhttp.Handler) stdhttp.Handler {
	return []func(stdhttp.Handler) stdhttp.Handler{
		appmiddleware.EnrichDiscordMembership(application.Resolver),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func writeParameterError(w stdhttp.ResponseWriter, _ *stdhttp.Request, err error) {
	response.WriteError(w, stdhttp.StatusBadRequest, "INVALID_QUERY_FILTER", err.Error())
}
