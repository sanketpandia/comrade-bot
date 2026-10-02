package gameflights

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/metrics"
	"infinite-experiment/politburo/internal/transport/http/api/cachedresponse"
)

type cacheStub struct {
	flights  domainflights.Snapshot
	records  map[string]domainflights.Flight
	names    []string
	namesErr error
	err      error
}

func (s cacheStub) GetJSON(_ context.Context, key string, destination any) error {
	switch key {
	case cache.KeySessionNames:
		if s.namesErr != nil {
			return s.namesErr
		}
		*(destination.(*[]string)) = s.names
		return nil
	case cache.KeyActiveFlights("casual"):
		if s.err != nil {
			return s.err
		}
		*(destination.(*domainflights.Snapshot)) = s.flights
		return nil
	default:
		if s.records != nil {
			for id, flight := range s.records {
				if key == cache.KeyFlightRecord(id) {
					*(destination.(*domainflights.Flight)) = flight
					return nil
				}
			}
		}
		return cache.ErrMiss
	}
}

func flightsFixture(lastCached time.Time, flights ...domainflights.Flight) (domainflights.Snapshot, map[string]domainflights.Flight) {
	tracks := make(map[string]domainflights.FlightMotion, len(flights))
	records := make(map[string]domainflights.Flight, len(flights))
	for i := range flights {
		if flights[i].FlightID == "" {
			flights[i].FlightID = "c34118e7-cbdd-4e22-8751-0cda93e41d75"
		}
		id := flights[i].FlightID
		tracks[id] = domainflights.MotionFromFlight(flights[i])
		records[id] = flights[i]
	}
	return domainflights.Snapshot{LastCached: lastCached, Tracks: tracks}, records
}

func (cacheStub) SetJSON(context.Context, string, any, time.Duration) error { return nil }

var testFlightSecret = []byte("0123456789abcdef0123456789abcdef")

func testHandler(store cache.Store) *Handler {
	return NewHandler(store, testFlightSecret, metrics.NewRegistry())
}

func sampleFlight(state string) domainflights.Flight {
	return domainflights.Flight{
		FlightID:       "c34118e7-cbdd-4e22-8751-0cda93e41d75",
		Callsign:       "Swiss 39 Heavy",
		Latitude:       47.45,
		Longitude:      8.56,
		Heading:        329.2,
		NormalizedName: "casual",
		Normalized:     domainflights.Normalized{PilotState: state, Speed: "526 kts", VerticalSpeed: "0.0 ft/min", IsConnected: "disconnected"},
		PathSync:       &domainflights.PathSync{FPLSyncRequired: false},
	}
}

func defaultQuery(serverID string, pilotStates []string) Query {
	return Query{
		ServerID:    serverID,
		PilotStates: pilotStates,
		PageNumber:  domainflights.DefaultPageNumber,
		PageLength:  domainflights.DefaultPageLength,
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestGetTrimmedActiveFlightsReturnsMarkersWithoutPaging(t *testing.T) {
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
	recorder := httptest.NewRecorder()
	handler.GetTrimmedActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/trimmed?serverId=casual", nil), Query{ServerID: "casual"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data struct {
			Count  int `json:"count"`
			Result []struct {
				FlightID  string  `json:"flightId"`
				Callsign  string  `json:"callsign"`
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
				Heading   float64 `json:"heading"`
				UserID    string  `json:"userId"`
			} `json:"result"`
			Pagination *cachedresponse.Pagination `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Count != 2 || len(body.Data.Result) != 2 || body.Data.Pagination != nil {
		t.Fatalf("count=%d result=%d pagination=%#v", body.Data.Count, len(body.Data.Result), body.Data.Pagination)
	}
	marker := body.Data.Result[0]
	if marker.Callsign != "Swiss 39 Heavy" || marker.Latitude != 47.45 || marker.Longitude != 8.56 || marker.Heading != 329.2 {
		t.Fatalf("marker = %#v", marker)
	}
	if marker.UserID != "" {
		t.Fatalf("trimmed payload leaked userId: %#v", marker)
	}
	if marker.FlightID == sampleFlight(domainflights.PilotStateNameActive).FlightID {
		t.Fatal("trimmed flightId must be encrypted")
	}
	token, err := domainflights.NewTokens(testFlightSecret).Decode(marker.FlightID)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if token.FlightID != "c34118e7-cbdd-4e22-8751-0cda93e41d75" || token.ServerID != "casual" {
		t.Fatalf("token = %#v", token)
	}
}

func TestGetActiveFlightResolvesEncryptedMarker(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	flight := sampleFlight(domainflights.PilotStateNameActive)
	snapshot, records := flightsFixture(lastCached, flight)
	handler := testHandler(cacheStub{
		names:   []string{"casual"},
		flights: snapshot,
		records: records,
	})
	token, err := domainflights.NewTokens(testFlightSecret).Encode(domainflights.MarkerToken{
		FlightID: flight.FlightID,
		ServerID: "casual",
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.GetActiveFlight(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/detail?flightId="+url.QueryEscape(token), nil), token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data struct {
			Result domainflights.Flight `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Result.FlightID != flight.FlightID || body.Data.Result.Callsign != flight.Callsign {
		t.Fatalf("result = %#v", body.Data.Result)
	}
}

func TestGetActiveFlightRejectsInvalidToken(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlight(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/detail?flightId=not-a-token", nil), "not-a-token")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}
