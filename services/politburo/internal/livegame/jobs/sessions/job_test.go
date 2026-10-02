package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type sessionsClientStub struct {
	sessions []infiniteflight.Session
	err      error
}

type cacheWrite struct {
	key   string
	value any
	ttl   time.Duration
}

type cacheStub struct {
	existing *gamesessions.Snapshot
	getError error
	writes   []cacheWrite
	setError error
}

func (s *cacheStub) GetJSON(_ context.Context, _ string, destination any) error {
	if s.getError != nil {
		return s.getError
	}
	if s.existing == nil {
		return cache.ErrMiss
	}
	*(destination.(*gamesessions.Snapshot)) = *s.existing
	return nil
}

func (s *cacheStub) SetJSON(_ context.Context, key string, value any, ttl time.Duration) error {
	if s.setError != nil {
		return s.setError
	}
	s.writes = append(s.writes, cacheWrite{key: key, value: value, ttl: ttl})
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

func (s sessionsClientStub) GetSessions(context.Context) ([]infiniteflight.Session, error) {
	return s.sessions, s.err
}

func TestJobRunFetchesSessions(t *testing.T) {
	cacheStore := &cacheStub{}
	job := New(sessionsClientStub{sessions: []infiniteflight.Session{{ID: "casual-id", Name: "Casual", UserCount: 10}}}, cacheStore)
	now := time.Date(2026, time.August, 14, 10, 30, 0, 0, time.FixedZone("test", 5*60*60+30*60))
	job.now = func() time.Time { return now }

	if job.Name() != "infinite-flight-sessions" {
		t.Fatalf("Name() = %q", job.Name())
	}
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	active, ok := cacheStore.write(cache.KeyActiveSessions)
	if !ok {
		t.Fatalf("missing write for %q; writes = %#v", cache.KeyActiveSessions, cacheStore.writes)
	}
	if active.ttl != gamesessions.CacheTTL {
		t.Fatalf("cache TTL = %s", active.ttl)
	}
	snapshot, ok := active.value.(gamesessions.Snapshot)
	if !ok || len(snapshot.Result) != 1 || !snapshot.LastCached.Equal(now.UTC()) {
		t.Fatalf("cached snapshot = %#v", active.value)
	}
	if !snapshot.Result[0].Timestamp.Equal(now.UTC()) {
		t.Fatalf("session timestamp = %s, want %s", snapshot.Result[0].Timestamp, now.UTC())
	}
	if snapshot.Result[0].NormalizedName != "casual" {
		t.Fatalf("normalized name = %q, want casual", snapshot.Result[0].NormalizedName)
	}
	encoded, err := json.Marshal(snapshot.Result[0])
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if !strings.Contains(string(encoded), `"timestamp":"2026-08-14T05:00:00Z"`) {
		t.Fatalf("session JSON = %s, want ISO 8601 UTC timestamp", encoded)
	}
	names, ok := cacheStore.write(cache.KeySessionNames)
	if !ok {
		t.Fatalf("missing write for %q", cache.KeySessionNames)
	}
	sessionNames, ok := names.value.([]string)
	if !ok || len(sessionNames) != 1 || sessionNames[0] != "casual" {
		t.Fatalf("session names = %#v", names.value)
	}
}

func TestJobRunKeepsOnlyLatestSessionPerID(t *testing.T) {
	refreshedAt := time.Date(2026, time.August, 14, 19, 5, 0, 0, time.UTC)
	cacheStore := &cacheStub{existing: &gamesessions.Snapshot{
		Result:     []infiniteflight.Session{{ID: "casual", UserCount: 100}},
		LastCached: refreshedAt.Add(-gamesessions.RefreshInterval),
	}}
	job := New(sessionsClientStub{sessions: []infiniteflight.Session{
		{ID: "casual", Name: "Casual", UserCount: 277},
		{ID: "training", Name: "Training", UserCount: 487},
		{ID: "casual", Name: "Casual", UserCount: 281},
	}}, cacheStore)
	job.now = func() time.Time { return refreshedAt }

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	active, ok := cacheStore.write(cache.KeyActiveSessions)
	if !ok {
		t.Fatalf("missing write for %q", cache.KeyActiveSessions)
	}
	snapshot := active.value.(gamesessions.Snapshot)
	if len(snapshot.Result) != 2 {
		t.Fatalf("current result length = %d, want 2", len(snapshot.Result))
	}
	if snapshot.Result[0].UserCount != 281 {
		t.Fatalf("latest casual user count = %d, want 281", snapshot.Result[0].UserCount)
	}
	for _, session := range snapshot.Result {
		if !session.Timestamp.Equal(refreshedAt) || session.NormalizedName == "" {
			t.Fatalf("session was not enriched consistently: %#v", session)
		}
	}
}

func TestJobRunSucceedsAfterCacheReadFailure(t *testing.T) {
	cacheStore := &cacheStub{getError: errors.New("redis unavailable")}
	job := New(sessionsClientStub{sessions: []infiniteflight.Session{{Name: "Casual"}}}, cacheStore)

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, ok := cacheStore.write(cache.KeyActiveSessions); !ok {
		t.Fatalf("missing write for %q", cache.KeyActiveSessions)
	}
}

func TestJobRunWrapsClientError(t *testing.T) {
	job := New(sessionsClientStub{err: errors.New("unavailable")}, &cacheStub{})

	err := job.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "refresh sessions: unavailable") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestJobRunWrapsCacheError(t *testing.T) {
	job := New(sessionsClientStub{}, &cacheStub{setError: errors.New("unavailable")})

	err := job.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cache sessions: unavailable") {
		t.Fatalf("Run() error = %v", err)
	}
}
