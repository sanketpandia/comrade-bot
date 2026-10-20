package vaflightindex

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/metrics"
)

type cacheStub struct {
	data   map[string]any
	writes []struct {
		key   string
		value any
		ttl   time.Duration
	}
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
	s.writes = append(s.writes, struct {
		key   string
		value any
		ttl   time.Duration
	}{key: key, value: value, ttl: ttl})
	if s.data == nil {
		s.data = map[string]any{}
	}
	s.data[key] = value
	return nil
}

func TestRunWritesVAKeysForMatchedFlights(t *testing.T) {
	store := &cacheStub{data: map[string]any{
		cache.KeyActiveSessions: gamesessions.Snapshot{
			Result: []infiniteflight.Session{{ID: "s1", NormalizedName: "expert"}},
		},
		cache.KeyActiveFlights("expert"): gameflights.Snapshot{
			Tracks: map[string]gameflights.FlightMotion{"f1": {}},
		},
		cache.KeyFlightRecord("f1"): gameflights.Flight{
			FlightID:               "f1",
			MatchedVirtualAirlines: []gameflights.VirtualAirlineMatch{{VAID: "va-1", VACode: "TST"}},
		},
	}}
	job := New(store, metrics.NewRegistry())
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(store.writes) != 1 {
		t.Fatalf("writes = %d", len(store.writes))
	}
	if store.writes[0].key != cache.KeyVAActiveFlights("va-1") {
		t.Fatalf("key = %q", store.writes[0].key)
	}
	ids, ok := store.writes[0].value.([]string)
	if !ok || len(ids) != 1 || ids[0] != "f1" {
		t.Fatalf("value = %#v", store.writes[0].value)
	}
	if store.writes[0].ttl != cache.ActiveFlightsTTL {
		t.Fatalf("ttl = %v", store.writes[0].ttl)
	}
}
