package flights

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	"infinite-experiment/politburo/internal/metrics"
)

type flightsClientStub struct {
	bySession map[string][]infiniteflight.Flight
	errByID   map[string]error
}

func (s flightsClientStub) GetSessionFlights(_ context.Context, sessionID string) ([]infiniteflight.Flight, error) {
	if err := s.errByID[sessionID]; err != nil {
		return nil, err
	}
	return s.bySession[sessionID], nil
}

type cacheWrite struct {
	key   string
	value any
	ttl   time.Duration
}

type cacheStub struct {
	data     map[string]any
	writes   []cacheWrite
	setError error
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
	if s.setError != nil {
		return s.setError
	}
	s.writes = append(s.writes, cacheWrite{key: key, value: value, ttl: ttl})
	if s.data == nil {
		s.data = map[string]any{}
	}
	s.data[key] = value
	return nil
}

func (s *cacheStub) write(key string) (cacheWrite, bool) {
	for i := len(s.writes) - 1; i >= 0; i-- {
		if s.writes[i].key == key {
			return s.writes[i], true
		}
	}
	return cacheWrite{}, false
}

func (s *cacheStub) writesTo(key string) int {
	count := 0
	for _, write := range s.writes {
		if write.key == key {
			count++
		}
	}
	return count
}

func withSessions(sessions []infiniteflight.Session) map[string]any {
	return map[string]any{
		cache.KeyActiveSessions: gamesessions.Snapshot{Result: sessions, LastCached: time.Now().UTC()},
	}
}

func testLookup(liveryID string, aircraftName, liveryName string) *gameliveries.Lookup {
	if liveryID == "" {
		return gameliveries.NewStaticLookup(nil, nil)
	}
	return gameliveries.NewStaticLookup(map[string]gameliveries.AircraftLivery{
		liveryID: {
			LiveryID: liveryID, DisplayAircraftName: aircraftName, DisplayLiveryName: liveryName,
		},
	}, nil)
}

func TestJobRunSkipsWhenSessionsMissing(t *testing.T) {
	store := &cacheStub{data: map[string]any{}}
	job := New(flightsClientStub{}, store, gameliveries.NewLookup(nil), metrics.NewRegistry())
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(store.writes) != 0 {
		t.Fatalf("writes = %#v", store.writes)
	}
}

func TestJobRunCachesMotionSnapshot(t *testing.T) {
	liveryID := "df597aaf-456c-4878-9d84-45201f2aae74"
	store := &cacheStub{data: withSessions([]infiniteflight.Session{{ID: "session-1", Name: "Casual", NormalizedName: "casual"}})}
	job := New(flightsClientStub{bySession: map[string][]infiniteflight.Flight{
		"session-1": {{FlightID: "f1", Callsign: "Swiss 39 Heavy", Speed: 525.6, LiveryID: liveryID, LastReport: "2026-08-15 05:09:53Z", PilotState: 3}},
	}}, store, testLookup(liveryID, "A350", "Swiss"), metrics.NewRegistry())
	now := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	job.now = func() time.Time { return now }

	if job.Name() != "infinite-flight-flights" {
		t.Fatalf("Name() = %q", job.Name())
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	write, ok := store.write(cache.KeyActiveFlights("casual"))
	if !ok {
		t.Fatalf("missing write; writes = %#v", store.writes)
	}
	if write.ttl != gameflights.GameActiveFlightTTL {
		t.Fatalf("ttl = %s", write.ttl)
	}
	snapshot := write.value.(gameflights.Snapshot)
	if !snapshot.LastCached.Equal(now) || len(snapshot.Tracks) != 1 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	motion := snapshot.Tracks["f1"]
	if motion.Speed != 526 {
		t.Fatalf("motion = %#v", motion)
	}
	if store.writesTo(cache.KeyFlightRecord("f1")) != 0 {
		t.Fatal("flight record should not be written while EnableFlightRecordWrites is false")
	}
}

func TestJobRunFastPathOnSecondTick(t *testing.T) {
	liveryID := "df597aaf-456c-4878-9d84-45201f2aae74"
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, LiveryID: liveryID,
		PilotState: gameflights.PilotStateInBackground, LastReport: "2026-08-15 05:09:53Z",
	}
	motion := gameflights.MotionFromUpstream(upstream)
	prior := gameflights.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, AircraftName: "A350", LiveryName: "Swiss",
		PathSync: &gameflights.PathSync{},
	}
	now := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	store := &cacheStub{data: withSessions([]infiniteflight.Session{{ID: "session-1", NormalizedName: "casual"}})}
	store.data[cache.KeyActiveFlights("casual")] = gameflights.Snapshot{
		LastCached: now.Add(-time.Minute),
		Tracks:     map[string]gameflights.FlightMotion{"f1": motion},
	}
	store.data[cache.KeyFlightRecord("f1")] = prior

	job := New(flightsClientStub{bySession: map[string][]infiniteflight.Flight{"session-1": {upstream}}}, store, testLookup(liveryID, "A350", "Swiss"), metrics.NewRegistry())
	job.now = func() time.Time { return now }

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	write, ok := store.write(cache.KeyActiveFlights("casual"))
	if !ok {
		t.Fatal("missing motion snapshot write")
	}
	if len(write.value.(gameflights.Snapshot).Tracks) != 1 {
		t.Fatalf("tracks = %#v", write.value)
	}
}

func TestJobRunContinuesAfterSessionError(t *testing.T) {
	store := &cacheStub{data: withSessions([]infiniteflight.Session{
		{ID: "bad", NormalizedName: "expert"},
		{ID: "good", NormalizedName: "casual"},
	})}
	job := New(flightsClientStub{
		bySession: map[string][]infiniteflight.Flight{"good": {{FlightID: "f1", Callsign: "ok", LastReport: "2026-08-15 05:09:53Z"}}},
		errByID:   map[string]error{"bad": errors.New("upstream")},
	}, store, gameliveries.NewLookup(nil), metrics.NewRegistry())
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, ok := store.write(cache.KeyActiveFlights("expert")); ok {
		t.Fatalf("expert snapshot should be left in place")
	}
	if _, ok := store.write(cache.KeyActiveFlights("casual")); !ok {
		t.Fatalf("missing casual snapshot; writes = %#v", store.writes)
	}
}

func TestJobRunWarnsAtCap(t *testing.T) {
	upstream := make([]infiniteflight.Flight, gameflights.MaxFlightsPerRequest)
	for i := range upstream {
		upstream[i].FlightID = "flight-" + strconv.Itoa(i)
		upstream[i].LastReport = "2026-08-15 05:09:53Z"
	}
	store := &cacheStub{data: withSessions([]infiniteflight.Session{{ID: "session-1", NormalizedName: "casual"}})}
	job := New(flightsClientStub{bySession: map[string][]infiniteflight.Flight{"session-1": upstream}}, store, gameliveries.NewLookup(nil), metrics.NewRegistry())
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	write, ok := store.write(cache.KeyActiveFlights("casual"))
	if !ok {
		t.Fatalf("missing write")
	}
	snapshot := write.value.(gameflights.Snapshot)
	if len(snapshot.Tracks) != gameflights.MaxFlightsPerRequest {
		t.Fatalf("tracks length = %d", len(snapshot.Tracks))
	}
}
