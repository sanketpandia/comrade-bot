package vaflightindex

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

type cacheStub struct {
	data   map[string]any
	writes map[string]cacheWrite
}

type cacheWrite struct {
	value any
	ttl   time.Duration
}

func (s *cacheStub) GetJSON(_ context.Context, key string, destination any) error {
	value, ok := s.data[key]
	if !ok {
		return cache.ErrMiss
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, destination)
}

func (s *cacheStub) SetJSON(_ context.Context, key string, value any, ttl time.Duration) error {
	if s.writes == nil {
		s.writes = map[string]cacheWrite{}
	}
	s.writes[key] = cacheWrite{value: value, ttl: ttl}
	if s.data == nil {
		s.data = map[string]any{}
	}
	s.data[key] = value
	return nil
}

type vaListerStub []virtualairlines.VirtualAirline

func (s vaListerStub) ListActive(context.Context) ([]virtualairlines.VirtualAirline, error) {
	return s, nil
}

func liveStore(record gameflights.Record) *cacheStub {
	flightID := record.Raw.FlightID
	return &cacheStub{data: map[string]any{
		cache.KeyActiveSessions: gamesessions.Snapshot{
			Result: []infiniteflight.Session{{ID: "s1", NormalizedName: "expert"}},
		},
		cache.KeyActiveFlights("expert"): gameflights.Snapshot{
			Tracks: map[string]gameflights.FlightMotion{flightID: {}},
		},
		cache.KeyFlightRecord(flightID): record,
	}}
}

var twoVAs = vaListerStub{{ID: "va-1", Code: "TST"}, {ID: "va-2", Code: "idle"}}

func TestRunWritesCodeIndexAndListsForEveryActiveVA(t *testing.T) {
	store := liveStore(gameflights.Record{
		Raw: gameflights.Observation{FlightID: "f1"},
		Derived: gameflights.Derived{
			MatchedVirtualAirlines: []virtualairlines.VirtualAirlineMatch{{VAID: "va-1", VACode: "TST"}},
		},
	})
	if err := New(store, twoVAs, metrics.NewRegistry()).Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	codes, ok := store.writes[cache.KeyVACodeIndex]
	if !ok || codes.ttl != cache.ActiveFlightsTTL {
		t.Fatalf("code index write = %#v", codes)
	}
	codeMap := codes.value.(map[string]string)
	if codeMap["TST"] != "va-1" || codeMap["IDLE"] != "va-2" {
		t.Fatalf("code index = %#v", codeMap)
	}

	matched := store.writes[cache.KeyVAActiveFlights("va-1")]
	if ids := matched.value.([]string); len(ids) != 1 || ids[0] != "f1" || matched.ttl != cache.ActiveFlightsTTL {
		t.Fatalf("va-1 write = %#v", matched)
	}
	idle, ok := store.writes[cache.KeyVAActiveFlights("va-2")]
	if !ok || len(idle.value.([]string)) != 0 {
		t.Fatalf("va-2 write = %#v", idle)
	}
}

func TestRunGaugeReportsZeroWhenVALosesMatches(t *testing.T) {
	record := gameflights.Record{
		Raw: gameflights.Observation{FlightID: "f1"},
		Derived: gameflights.Derived{
			MatchedVirtualAirlines: []virtualairlines.VirtualAirlineMatch{{VAID: "va-1", VACode: "TST"}},
		},
	}
	store := liveStore(record)
	registry := metrics.NewRegistry()
	job := New(store, vaListerStub{{ID: "va-1", Code: "TST"}}, registry)
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if got := testutil.ToFloat64(registry.VAActiveFlights.WithLabelValues("va-1")); got != 1 {
		t.Fatalf("gauge after match = %v", got)
	}

	record.Derived.MatchedVirtualAirlines = nil
	store.data[cache.KeyFlightRecord("f1")] = record
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if got := testutil.ToFloat64(registry.VAActiveFlights.WithLabelValues("va-1")); got != 0 {
		t.Fatalf("gauge after VA lost matches = %v", got)
	}
}

func TestRunDropsGaugeForDeactivatedVA(t *testing.T) {
	store := liveStore(gameflights.Record{Raw: gameflights.Observation{FlightID: "f1"}})
	registry := metrics.NewRegistry()
	if err := New(store, twoVAs, registry).Run(context.Background()); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if err := New(store, vaListerStub{{ID: "va-1", Code: "TST"}}, registry).Run(context.Background()); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if got := testutil.CollectAndCount(registry.VAActiveFlights); got != 1 {
		t.Fatalf("series = %d, want only active VA", got)
	}
}
