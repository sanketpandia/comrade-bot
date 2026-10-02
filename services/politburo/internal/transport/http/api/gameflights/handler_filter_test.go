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

func TestGetActiveFlightsFiltersByPilotState(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	active := sampleFlight(domainflights.PilotStateNameActive)
	bg := sampleFlight(domainflights.PilotStateNameInBackground)
	bg.FlightID = "flight-bg"
	snapshot, records := flightsFixture(lastCached, active, bg)
	handler := testHandler(cacheStub{
		names:   []string{"casual"},
		flights: snapshot,
		records: records,
	})
	states := []politburoapi.PilotStateName{"active"}
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?pilotState=active", nil), "casual", politburoapi.GetActiveFlightsParams{
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
	if len(body.Data.Result) != 1 || body.Data.Result[0].FlightID != active.FlightID {
		t.Fatalf("result = %#v", body.Data.Result)
	}
}
