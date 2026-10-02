package gamesessions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	domainsessions "infinite-experiment/politburo/internal/livegame/sessions"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

type cacheStub struct {
	snapshot domainsessions.Snapshot
	err      error
}

func (s cacheStub) GetJSON(_ context.Context, key string, destination any) error {
	if key != cache.KeyActiveSessions {
		return errors.New("unexpected cache key")
	}
	if s.err != nil {
		return s.err
	}
	*(destination.(*domainsessions.Snapshot)) = s.snapshot
	return nil
}

func (cacheStub) SetJSON(context.Context, string, any, time.Duration) error { return nil }

func TestGetActiveSessionsReturnsCachedResponse(t *testing.T) {
	lastCached := time.Date(2026, time.August, 14, 5, 0, 0, 123, time.UTC)
	handler := NewHandler(cacheStub{snapshot: domainsessions.Snapshot{
		Result: []infiniteflight.Session{{
			ID: "8c772474-bb70-4294-ad40-09f8cbf3b289", Name: "Casual",
			NormalizedName: "casual", UserCount: 42, Type: 1,
		}},
		LastCached: lastCached,
	}})
	recorder := httptest.NewRecorder()
	handler.GetActiveSessions(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/sessions/active", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data struct {
			Result []ActiveSession `json:"result"`
			Meta   struct {
				LastCached          time.Time `json:"lastCached"`
				RefreshIntervalMins int       `json:"refreshIntervalMins"`
			} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.Result) != 1 {
		t.Fatalf("result = %#v", body.Data.Result)
	}
	row := body.Data.Result[0]
	if row.NormalizedName != "casual" || row.UserCount != 42 || row.Type != 1 {
		t.Fatalf("result row = %#v", row)
	}
	if !body.Data.Meta.LastCached.Equal(lastCached) || body.Data.Meta.RefreshIntervalMins != 5 {
		t.Fatalf("meta = %#v", body.Data.Meta)
	}
	if strings.Contains(recorder.Body.String(), `"history"`) {
		t.Fatalf("response should not include history: %s", recorder.Body.String())
	}
}

func TestGetActiveSessionsReturnsServiceUnavailableOnCacheMiss(t *testing.T) {
	handler := NewHandler(cacheStub{err: cache.ErrMiss})
	recorder := httptest.NewRecorder()
	handler.GetActiveSessions(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/sessions/active", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}

func TestGetActiveSessionsRejectsSnapshotWithoutTimestamp(t *testing.T) {
	handler := NewHandler(cacheStub{snapshot: domainsessions.Snapshot{}})
	recorder := httptest.NewRecorder()
	handler.GetActiveSessions(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/sessions/active", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
