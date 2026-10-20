package gameflights

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
	"infinite-experiment/politburo/internal/community/virtualairlines"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
)

type vaCodeStub struct {
	byCode map[string]*virtualairlines.VirtualAirline
}

func (s vaCodeStub) GetByCode(_ context.Context, code string) (*virtualairlines.VirtualAirline, error) {
	if s.byCode == nil {
		return nil, nil
	}
	return s.byCode[code], nil
}

func TestGetActiveFlightsFiltersByVACode(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	active := sampleFlight(domainflights.PilotStateNameActive)
	active.FlightID = "flight-active"
	bg := sampleFlight(domainflights.PilotStateNameInBackground)
	bg.FlightID = "flight-bg"
	other := sampleFlight(domainflights.PilotStateNameActive)
	other.FlightID = "flight-other"
	snapshot, records := flightsFixture(lastCached, active, bg, other)

	handler := testHandlerWithVA(cacheStub{
		names:   []string{"casual"},
		flights: snapshot,
		records: records,
		vaIndexes: map[string][]string{
			"va-1": {"flight-active", "flight-bg", "not-on-server"},
		},
	}, vaCodeStub{byCode: map[string]*virtualairlines.VirtualAirline{
		"TEST": {ID: "va-1", Code: "TEST"},
	}})

	vaCode := "test"
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?vaCode=test", nil), "casual", politburoapi.GetActiveFlightsParams{
		VaCode: &vaCode,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data struct {
			Result []activeFlightTrack `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Result) != 2 {
		t.Fatalf("result = %#v", body.Data.Result)
	}
	ids := []string{body.Data.Result[0].FlightID, body.Data.Result[1].FlightID}
	if ids[0] != "flight-active" || ids[1] != "flight-bg" {
		t.Fatalf("sorted ids = %v", ids)
	}
}

func TestGetActiveFlightsVACodeWithPilotState(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	active := sampleFlight(domainflights.PilotStateNameActive)
	active.FlightID = "flight-active"
	bg := sampleFlight(domainflights.PilotStateNameInBackground)
	bg.FlightID = "flight-bg"
	snapshot, records := flightsFixture(lastCached, active, bg)

	handler := testHandlerWithVA(cacheStub{
		names:     []string{"casual"},
		flights:   snapshot,
		records:   records,
		vaIndexes: map[string][]string{"va-1": {"flight-active", "flight-bg"}},
	}, vaCodeStub{byCode: map[string]*virtualairlines.VirtualAirline{
		"TEST": {ID: "va-1", Code: "TEST"},
	}})

	vaCode := "TEST"
	states := []politburoapi.PilotStateName{"active"}
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?vaCode=TEST&pilotState=active", nil), "casual", politburoapi.GetActiveFlightsParams{
		VaCode:     &vaCode,
		PilotState: &states,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data struct {
			Result []activeFlightTrack `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Result) != 1 || body.Data.Result[0].FlightID != "flight-active" {
		t.Fatalf("result = %#v", body.Data.Result)
	}
}

func TestGetActiveFlightsUnknownVACode(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	snapshot, records := flightsFixture(lastCached, sampleFlight(domainflights.PilotStateNameActive))
	handler := testHandlerWithVA(cacheStub{
		names:   []string{"casual"},
		flights: snapshot,
		records: records,
	}, vaCodeStub{byCode: map[string]*virtualairlines.VirtualAirline{}})

	vaCode := "NOPE"
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?vaCode=NOPE", nil), "casual", politburoapi.GetActiveFlightsParams{
		VaCode: &vaCode,
	})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}
