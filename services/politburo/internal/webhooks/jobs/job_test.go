package jobs

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/webhooks/dispatch"
	"infinite-experiment/politburo/internal/webhooks/features/activeflights"
	"infinite-experiment/politburo/internal/webhooks/runlog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRunDispatchesDueFeaturesOnly(t *testing.T) {
	raw := json.RawMessage(`{
		"active_flights": {"url":"https://discord.com/api/webhooks/1/token","frequency":"30m","filters":{}},
		"hourly": {"url":"https://discord.com/api/webhooks/1/token","frequency":"60m","filters":{}},
		"mystery": {"url":"https://discord.com/api/webhooks/1/token","frequency":"30m","filters":{}}
	}`)
	loader := &stubLoader{rows: []virtualairlines.WebhookDispatchRow{{
		VAID: "va-1", VACode: "TEST", Webhooks: raw, PreferredServers: []string{"expert"},
	}}}
	handler := &stubHandler{result: dispatch.Result{Outcome: dispatch.OutcomeSuccess, DiscordMessages: 1}}
	registry := dispatch.NewRegistry()
	registry.Register(activeflights.FeatureKey, handler)
	runs := &stubRuns{}
	reg := metrics.NewRegistry()
	job := New(loader, registry, runs, reg)
	job.now = func() time.Time { return time.Date(2026, 10, 20, 14, 30, 0, 0, time.UTC) }

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	handler.mu.Lock()
	keys := append([]string(nil), handler.keys...)
	handler.mu.Unlock()
	if len(keys) != 1 || keys[0] != activeflights.FeatureKey {
		t.Fatalf("handler keys = %#v", keys)
	}

	runs.mu.Lock()
	logged := append([]runlog.Row(nil), runs.rows...)
	runs.mu.Unlock()
	if len(logged) != 2 {
		t.Fatalf("logged runs = %+v", logged)
	}
	outcomes := map[string]runlog.Row{}
	for _, row := range logged {
		outcomes[row.WebhookType] = row
	}
	if _, ok := outcomes["hourly"]; ok {
		t.Fatal("60m feature should not run at minute 30")
	}
	if outcomes[activeflights.FeatureKey].Outcome != dispatch.OutcomeSuccess || outcomes[activeflights.FeatureKey].DiscordMessages != 1 {
		t.Fatalf("active_flights = %+v", outcomes[activeflights.FeatureKey])
	}
	if outcomes["mystery"].Outcome != dispatch.OutcomeSkipped || outcomes["mystery"].SkipReason != dispatch.SkipUnknownFeature {
		t.Fatalf("mystery = %+v", outcomes["mystery"])
	}
	if got := testutil.ToFloat64(reg.WebhookDispatchTotal.WithLabelValues(activeflights.FeatureKey, dispatch.OutcomeSuccess)); got != 1 {
		t.Fatalf("success dispatches = %v", got)
	}
	if got := testutil.ToFloat64(reg.WebhookDispatchTotal.WithLabelValues("mystery", dispatch.OutcomeSkipped)); got != 1 {
		t.Fatalf("skipped dispatches = %v", got)
	}
}

func TestDispatchMetrics(t *testing.T) {
	cases := []struct {
		name    string
		result  dispatch.Result
		outcome string
	}{
		{name: "success", result: dispatch.Result{Outcome: dispatch.OutcomeSuccess, DiscordMessages: 2}, outcome: dispatch.OutcomeSuccess},
		{name: "skipped", result: dispatch.Result{Outcome: dispatch.OutcomeSkipped, SkipReason: dispatch.SkipNoPreferredServers}, outcome: dispatch.OutcomeSkipped},
		{name: "error", result: dispatch.Result{Outcome: dispatch.OutcomeError, ErrorMessage: "discord webhook post: status 500"}, outcome: dispatch.OutcomeError},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			raw := json.RawMessage(`{"active_flights":{"url":"https://discord.com/api/webhooks/1/token","frequency":"30m","filters":{}}}`)
			loader := &stubLoader{rows: []virtualairlines.WebhookDispatchRow{{VAID: "va-1", Webhooks: raw}}}
			registry := dispatch.NewRegistry()
			registry.Register(activeflights.FeatureKey, &stubHandler{result: tt.result})
			reg := metrics.NewRegistry()
			job := New(loader, registry, &stubRuns{}, reg)
			job.now = func() time.Time { return time.Date(2026, 10, 20, 14, 0, 0, 0, time.UTC) }
			if err := job.Run(context.Background()); err != nil {
				t.Fatalf("Run() = %v", err)
			}
			if got := testutil.ToFloat64(reg.WebhookDispatchTotal.WithLabelValues(activeflights.FeatureKey, tt.outcome)); got != 1 {
				t.Fatalf("dispatch_total = %v", got)
			}
			observer := reg.WebhookDispatchDuration.WithLabelValues(activeflights.FeatureKey)
			metric, ok := observer.(prometheus.Metric)
			if !ok {
				t.Fatal("duration observer is not a metric")
			}
			var dtoMetric dto.Metric
			if err := metric.Write(&dtoMetric); err != nil {
				t.Fatal(err)
			}
			if dtoMetric.GetHistogram().GetSampleCount() != 1 {
				t.Fatalf("histogram samples = %d", dtoMetric.GetHistogram().GetSampleCount())
			}
		})
	}
}

type stubLoader struct {
	rows []virtualairlines.WebhookDispatchRow
}

func (s *stubLoader) LoadWebhookDispatchRows(ctx context.Context) ([]virtualairlines.WebhookDispatchRow, error) {
	return s.rows, nil
}

type stubHandler struct {
	mu     sync.Mutex
	keys   []string
	result dispatch.Result
}

func (s *stubHandler) Handle(ctx context.Context, item dispatch.Item) dispatch.Result {
	s.mu.Lock()
	s.keys = append(s.keys, item.WebhookType)
	s.mu.Unlock()
	return s.result
}

type stubRuns struct {
	mu   sync.Mutex
	rows []runlog.Row
}

func (s *stubRuns) Insert(ctx context.Context, row runlog.Row) error {
	s.mu.Lock()
	s.rows = append(s.rows, row)
	s.mu.Unlock()
	return nil
}
