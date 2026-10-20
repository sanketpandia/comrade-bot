package webhooks

import (
	"infinite-experiment/politburo/internal/cache"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/webhooks/discord"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	"infinite-experiment/politburo/internal/webhooks/features/activeflights"
)

// NewRegistry registers all webhook feature handlers for dispatch.
func NewRegistry(cacheStore cache.Store, metricsRegistry *metrics.Registry) *dispatch.Registry {
	registry := dispatch.NewRegistry()
	registry.Register(activeflights.FeatureKey, activeflights.New(
		gameflights.NewReader(cacheStore),
		gamesessions.NewReader(cacheStore),
		discord.NewClient(metricsRegistry),
		metricsRegistry,
	))
	return registry
}
