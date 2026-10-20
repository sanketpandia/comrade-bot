package gameflights

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"

	"github.com/google/uuid"
)

var testFlightUUID = uuid.MustParse("c34118e7-cbdd-4e22-8751-0cda93e41d75")

func TestGetActiveFlightReturnsRecord(t *testing.T) {
	lastCached := time.Date(2026, 8, 15, 5, 10, 0, 0, time.UTC)
	flight := sampleFlight("active")
	flight.LastReport = lastCached
	_, records := flightsFixture(lastCached, flight)
	handler := testHandler(cacheStub{records: records})

	recorder := httptest.NewRecorder()
	handler.GetActiveFlight(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flight/active/"+testFlightUUID.String(), nil), testFlightUUID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var body politburoapi.ActiveFlightRecordResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.Callsign != "Swiss 39 Heavy" || body.Data.Normalized.PilotState != politburoapi.Active {
		t.Fatalf("data = %#v", body.Data)
	}
}

func TestGetActiveFlightReturnsNotFoundOnCacheMiss(t *testing.T) {
	handler := testHandler(cacheStub{})

	recorder := httptest.NewRecorder()
	handler.GetActiveFlight(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flight/active/"+testFlightUUID.String(), nil), testFlightUUID)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestGetActiveFlightReturnsInternalErrorOnCacheReadFailure(t *testing.T) {
	handler := testHandler(cacheStub{
		records: map[string]domainflights.Flight{testFlightUUID.String(): sampleFlight("active")},
		err:     errors.New("redis down"),
	})

	recorder := httptest.NewRecorder()
	handler.GetActiveFlight(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flight/active/"+testFlightUUID.String(), nil), testFlightUUID)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
