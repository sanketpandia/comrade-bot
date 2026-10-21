package gameflights

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
)

func getActiveFlightsWithVA(t *testing.T, store cacheStub, vaCode string, states ...politburoapi.PilotStateName) *httptest.ResponseRecorder {
	t.Helper()
	params := politburoapi.GetActiveFlightsParams{VaCode: &vaCode}
	if len(states) > 0 {
		params.PilotState = &states
	}
	recorder := httptest.NewRecorder()
	testHandler(store).GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil), "casual", params)
	return recorder
}

func decodeTrackIDs(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()
	var body struct {
		Data struct {
			Result []activeFlightTrack `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	ids := make([]string, len(body.Data.Result))
	for i, track := range body.Data.Result {
		ids[i] = track.FlightID
	}
	return ids
}

func TestGetActiveFlightsFiltersByVACode(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	active := sampleFlight(domainflights.PilotStateNameActive)
	active.Raw.FlightID = "flight-active"
	bg := sampleFlight(domainflights.PilotStateNameInBackground)
	bg.Raw.FlightID = "flight-bg"
	other := sampleFlight(domainflights.PilotStateNameActive)
	other.Raw.FlightID = "flight-other"
	snapshot, records := flightsFixture(lastCached, active, bg, other)

	recorder := getActiveFlightsWithVA(t, cacheStub{
		names:     []string{"casual"},
		flights:   snapshot,
		records:   records,
		vaCodes:   map[string]string{"TEST": "va-1"},
		vaIndexes: map[string][]string{"va-1": {"flight-active", "flight-bg", "not-on-server"}},
	}, " test ")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	ids := decodeTrackIDs(t, recorder)
	if len(ids) != 2 || ids[0] != "flight-active" || ids[1] != "flight-bg" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestGetActiveFlightsVACodeWithPilotState(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	active := sampleFlight(domainflights.PilotStateNameActive)
	active.Raw.FlightID = "flight-active"
	bg := sampleFlight(domainflights.PilotStateNameInBackground)
	bg.Raw.FlightID = "flight-bg"
	snapshot, records := flightsFixture(lastCached, active, bg)

	recorder := getActiveFlightsWithVA(t, cacheStub{
		names:     []string{"casual"},
		flights:   snapshot,
		records:   records,
		vaCodes:   map[string]string{"TEST": "va-1"},
		vaIndexes: map[string][]string{"va-1": {"flight-active", "flight-bg"}},
	}, "TEST", "active")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	if ids := decodeTrackIDs(t, recorder); len(ids) != 1 || ids[0] != "flight-active" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestGetActiveFlightsVACodeWithNoMatchedFlights(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	snapshot, records := flightsFixture(lastCached, sampleFlight(domainflights.PilotStateNameActive))
	recorder := getActiveFlightsWithVA(t, cacheStub{
		names:     []string{"casual"},
		flights:   snapshot,
		records:   records,
		vaCodes:   map[string]string{"TEST": "va-1"},
		vaIndexes: map[string][]string{"va-1": {}},
	}, "TEST")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	if ids := decodeTrackIDs(t, recorder); len(ids) != 0 {
		t.Fatalf("ids = %v", ids)
	}
}

func TestGetActiveFlightsUnknownVACode(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	snapshot, records := flightsFixture(lastCached, sampleFlight(domainflights.PilotStateNameActive))
	recorder := getActiveFlightsWithVA(t, cacheStub{
		names:   []string{"casual"},
		flights: snapshot,
		records: records,
		vaCodes: map[string]string{"TEST": "va-1"},
	}, "NOPE")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}

func TestGetActiveFlightsVACodeIndexMissIsUnavailable(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	snapshot, records := flightsFixture(lastCached, sampleFlight(domainflights.PilotStateNameActive))
	recorder := getActiveFlightsWithVA(t, cacheStub{
		names:   []string{"casual"},
		flights: snapshot,
		records: records,
	}, "TEST")
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}
