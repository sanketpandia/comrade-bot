package http

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/app"
	"infinite-experiment/politburo/internal/config"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/ui"

	"github.com/go-chi/chi/v5"
)

func TestRouterRouteTable(t *testing.T) {
	renderer, err := ui.NewRenderer()
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	application := &app.App{
		Config:    cfg,
		Metrics:   metrics.NewRegistry(),
		UI:        renderer,
		StartedAt: time.Now().UTC(),
	}

	r := NewServer(application).router()
	got := collectRoutes(r.(chi.Routes))
	sort.Strings(got)

	// Spot-check stable routes; full table guards against accidental drops.
	mustHave := []string{
		"GET /health/status",
		"POST /api/v1/signed-link",
		"GET /api/v1/user/status",
		"GET /auth/login",
		"GET /dashboard",
		"GET /maps/flights/active",
		"GET /metrics",
	}
	for _, want := range mustHave {
		if !contains(got, want) {
			t.Fatalf("route table missing %q (have %d routes)", want, len(got))
		}
	}
	if len(got) < 30 {
		t.Fatalf("route table too small: %d routes", len(got))
	}
}

func collectRoutes(r chi.Routes) []string {
	var out []string
	chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		out = append(out, method+" "+route)
		return nil
	})
	return out
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
