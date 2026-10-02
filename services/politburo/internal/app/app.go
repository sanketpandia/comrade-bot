package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"infinite-experiment/politburo/internal/access/apikeys"
	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/access/session"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/membership"
	"infinite-experiment/politburo/internal/community/membership/resolver"
	"infinite-experiment/politburo/internal/community/membership/roster"
	"infinite-experiment/politburo/internal/community/registration"
	"infinite-experiment/politburo/internal/community/users"
	"infinite-experiment/politburo/internal/community/users/status"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/community/virtualairlines/lookup"
	"infinite-experiment/politburo/internal/config"
	"infinite-experiment/politburo/internal/database"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/livegame/jobs"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/livegame/scheduler"
	"infinite-experiment/politburo/internal/logging"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/operations/operator"
	"infinite-experiment/politburo/internal/operations/reports"
	"infinite-experiment/politburo/internal/ui"
)

type App struct {
	Config    config.Config
	StartedAt time.Time
	DB        *sql.DB
	Cache     *cache.RedisStore
	APIKeys   *apikeys.Lookup
	Users     *users.Repository
	Sessions  *session.Service
	Tickets   *auth.Tickets
	Metrics   *metrics.Registry
	Scheduler *scheduler.Scheduler
	UI        *ui.Renderer

	Registration *registration.Service
	Membership   *membership.Service
	VAInit       *virtualairlines.Service
	Status       *status.Builder
	Reports      *reports.Repository
	Operator     *operator.Service
	Resolver         *resolver.Resolver
	VALookup         *lookup.VALookup
	VirtualAirlines  *virtualairlines.Repository
	IFUsers          infiniteflight.UsersClient
	LiveryLookup     *gameliveries.Lookup
	closeOnce        sync.Once
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logging.Init(cfg.Environment)
	metricsRegistry := metrics.NewRegistry()

	db, err := database.Open(ctx, cfg.Database.URL, cfg.Database.PingTimeout)
	if err != nil {
		return nil, err
	}

	cacheStore, err := cache.OpenRedis(ctx, cfg.Redis, metricsRegistry)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	renderer, err := ui.NewRenderer()
	if err != nil {
		_ = cacheStore.Close()
		_ = db.Close()
		return nil, fmt.Errorf("initialize UI assets: %w", err)
	}

	jobScheduler := scheduler.New(metricsRegistry)
	infiniteFlightClient, err := infiniteflight.NewClient(
		cfg.InfiniteFlight.BaseURL,
		cfg.InfiniteFlight.APIKey,
		cfg.InfiniteFlight.RequestTimeout,
	)
	if err != nil {
		_ = cacheStore.Close()
		_ = db.Close()
		return nil, err
	}
	liveryRepo := gameliveries.NewRepository(db)
	liveryLookup := gameliveries.NewLookup(liveryRepo)
	if err := liveryLookup.Reload(ctx); err != nil {
		slog.Warn("initial livery lookup reload failed; catalog may be empty until first sync", "error", err)
	}
	if err := jobs.Register(jobScheduler, infiniteFlightClient, cacheStore, db, liveryLookup, metricsRegistry); err != nil {
		_ = cacheStore.Close()
		_ = db.Close()
		return nil, fmt.Errorf("register jobs: %w", err)
	}

	userRepo := users.NewRepository(db)
	vaRepo := virtualairlines.NewRepository(db)
	membershipRepo := membership.NewRepository(db)
	reportRepo := reports.NewRepository(db)

	var ifUsers infiniteflight.UsersClient
	if cfg.InfiniteFlight.APIKey != "" {
		ifUsers = infiniteFlightClient
	}

	application := &App{
		Config: cfg, StartedAt: time.Now().UTC(), DB: db, Cache: cacheStore,
		APIKeys: apikeys.NewLookup(apikeys.NewRepository(db), cacheStore),
		Users:   userRepo, Sessions: session.NewService(cacheStore),
		Tickets: auth.NewTickets(cacheStore, cfg.Auth.SignedLinkSecret),
		Metrics: metricsRegistry, Scheduler: jobScheduler, UI: renderer,
		Registration: registration.NewService(userRepo, ifUsers),
		Membership:   membership.NewService(userRepo, membershipRepo, vaRepo, roster.Noop{}),
		VAInit:       virtualairlines.NewService(db, userRepo, vaRepo),
		Status:       status.NewBuilder(userRepo, membershipRepo, vaRepo),
		Reports:      reportRepo,
		Operator:     operator.NewService(db, userRepo, membershipRepo, reportRepo, vaRepo),
		Resolver:        resolver.NewResolver(userRepo, vaRepo, membershipRepo),
		VALookup:        lookup.NewVALookup(vaRepo),
		VirtualAirlines: vaRepo,
		IFUsers:      ifUsers,
		LiveryLookup: liveryLookup,
	}
	slog.Info("application initialized", "environment", cfg.Environment, "jobs_enabled", cfg.Jobs.Enabled)
	return application, nil
}

func (a *App) Close() {
	a.closeOnce.Do(func() {
		a.Scheduler.Stop()
		if err := a.Cache.Close(); err != nil {
			slog.Error("close Redis", "error", err)
		}
		if err := a.DB.Close(); err != nil {
			slog.Error("close database", "error", err)
		}
	})
}
