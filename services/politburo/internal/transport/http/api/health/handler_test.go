package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	gamesessions "infinite-experiment/politburo/internal/livegame/sessions"
)

type stubDB struct {
	err error
}

func (s stubDB) PingContext(context.Context) error {
	return s.err
}

type stubCache struct {
	pingErr  error
	getErr   error
	snapshot *gamesessions.Snapshot
}

func (s stubCache) Ping(context.Context) error {
	return s.pingErr
}

func (s stubCache) GetJSON(_ context.Context, key string, destination any) error {
	if key != cache.KeyActiveSessions {
		return errors.New("unexpected key")
	}
	if s.getErr != nil {
		return s.getErr
	}
	if s.snapshot == nil {
		return cache.ErrMiss
	}
	target, ok := destination.(*gamesessions.Snapshot)
	if !ok {
		return errors.New("unexpected destination type")
	}
	*target = *s.snapshot
	return nil
}

func TestStatusAllActive(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubDB{}, stubCache{}, false, time.Now().Add(-time.Second))
	recorder := httptest.NewRecorder()
	handler.GetStatus(recorder, httptest.NewRequest(http.MethodGet, "/health/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("body = %s, want ok status", body)
	}
	for _, want := range []string{`"database":"active"`, `"cache":"active"`, `"infinite-flight":"active"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestStatusCacheDown(t *testing.T) {
	t.Parallel()

	handler := NewHandler(stubDB{}, stubCache{pingErr: errors.New("redis down")}, false, time.Now())
	recorder := httptest.NewRecorder()
	handler.GetStatus(recorder, httptest.NewRequest(http.MethodGet, "/health/status", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"status":"down"`) {
		t.Fatalf("body = %s, want down status", body)
	}
	if !strings.Contains(body, `"database":"active"`) || !strings.Contains(body, `"cache":"down"`) {
		t.Fatalf("body = %s, want database active and cache down", body)
	}
}

func TestStatusSessionsStaleWhenJobsEnabled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 12, 10, 0, 0, time.UTC)
	handler := NewHandler(stubDB{}, stubCache{snapshot: &gamesessions.Snapshot{
		LastCached: now.Add(-gamesessions.RefreshInterval - time.Minute),
	}}, true, now)
	handler.now = func() time.Time { return now }

	recorder := httptest.NewRecorder()
	handler.GetStatus(recorder, httptest.NewRequest(http.MethodGet, "/health/status", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"infinite-flight":"down"`) {
		t.Fatalf("body = %s, want infinite-flight down", body)
	}
}

func TestStatusSessionsFreshWhenJobsEnabled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 12, 10, 0, 0, time.UTC)
	handler := NewHandler(stubDB{}, stubCache{snapshot: &gamesessions.Snapshot{
		LastCached: now.Add(-2 * time.Minute),
	}}, true, now)
	handler.now = func() time.Time { return now }

	recorder := httptest.NewRecorder()
	handler.GetStatus(recorder, httptest.NewRequest(http.MethodGet, "/health/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"infinite-flight":"active"`) {
		t.Fatalf("body = %s, want infinite-flight active", recorder.Body.String())
	}
}
