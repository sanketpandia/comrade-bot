// Package dispatch routes due webhook features through a bounded worker pool.
package dispatch

import (
	"context"
	"encoding/json"
	"log/slog"

	"golang.org/x/sync/errgroup"
)

const (
	DefaultWorkers = 8

	OutcomeSuccess = "success"
	OutcomeSkipped = "skipped"
	OutcomeError   = "error"

	SkipNoPreferredServers = "no_preferred_servers"
	SkipUnknownFeature     = "unknown_feature"
)

// Item is one due feature entry for a single VA.
type Item struct {
	VAID             string
	VACode           string
	WebhookType      string
	URL              string
	Filters          json.RawMessage
	PreferredServers []string
}

// Result is the transport outcome of one feature dispatch.
type Result struct {
	Outcome         string
	SkipReason      string
	DiscordMessages int
	ErrorMessage    string
}

// Handler runs one feature.
type Handler interface {
	Handle(ctx context.Context, item Item) Result
}

// Registry maps feature keys to handlers. A missing key is a skip, not a validation error.
type Registry struct {
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

func (r *Registry) Register(feature string, handler Handler) {
	r.handlers[feature] = handler
}

func (r *Registry) Handle(ctx context.Context, item Item) Result {
	handler, ok := r.handlers[item.WebhookType]
	if !ok {
		slog.Debug("skip unknown webhook feature", "va_id", item.VAID, "webhook_type", item.WebhookType)
		return Result{Outcome: OutcomeSkipped, SkipReason: SkipUnknownFeature}
	}
	return handler.Handle(ctx, item)
}

// Run calls fn for each index with at most workers concurrent calls.
// A non-nil error from fn cancels the remaining work.
func Run(ctx context.Context, workers int, n int, fn func(context.Context, int) error) error {
	if n == 0 {
		return nil
	}
	if workers < 1 {
		workers = DefaultWorkers
	}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(workers)
	for i := 0; i < n; i++ {
		i := i
		group.Go(func() error {
			return fn(ctx, i)
		})
	}
	return group.Wait()
}
