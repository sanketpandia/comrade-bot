package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"infinite-experiment/politburo/internal/apikeys"
	"infinite-experiment/politburo/internal/auth"
	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/config"
	"infinite-experiment/politburo/internal/database"
	"infinite-experiment/politburo/internal/identity"
	"infinite-experiment/politburo/internal/infiniteflight"
	"infinite-experiment/politburo/internal/jobs"
	"infinite-experiment/politburo/internal/logging"
	"infinite-experiment/politburo/internal/membership"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/operator"
	"infinite-experiment/politburo/internal/registration"
	"infinite-experiment/politburo/internal/reports"
	"infinite-experiment/politburo/internal/scheduler"
	"infinite-experiment/politburo/internal/session"
	"infinite-experiment/politburo/internal/ui"
	"infinite-experiment/politburo/internal/users"
	"infinite-experiment/politburo/internal/userstatus"
	"infinite-experiment/politburo/internal/va/roster"
	"infinite-experiment/politburo/internal/virtualairlines"
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
	Status       *userstatus.Builder
	Reports      *reports.Repository
	Operator     *operator.Service
	Resolver         *identity.Resolver
	VALookup         *identity.VALookup
	VirtualAirlines  *virtualairlines.Repository
	IFUsers          infiniteflight.UsersClient
	closeOnce    sync.Once
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
	if err := jobs.Register(jobScheduler, infiniteFlightClient, cacheStore); err != nil {
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
		Status:       userstatus.NewBuilder(userRepo, membershipRepo, vaRepo),
		Reports:      reportRepo,
		Operator:     operator.NewService(db, userRepo, membershipRepo, reportRepo, vaRepo),
		Resolver:        identity.NewResolver(userRepo, vaRepo, membershipRepo),
		VALookup:        identity.NewVALookup(vaRepo),
		VirtualAirlines: vaRepo,
		IFUsers:         ifUsers,
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
