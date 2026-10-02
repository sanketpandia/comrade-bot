package gameflights

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/cache"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
)

func motionSnapshot(lastCached time.Time, ids ...string) domainflights.Snapshot {
	tracks := make(map[string]domainflights.FlightMotion, len(ids))
	for i, id := range ids {
		tracks[id] = domainflights.FlightMotion{
			Latitude:  47.45 + float64(i)*0.01,
			Longitude: 8.56,
			Speed:     400 + i,
			Callsign:  "CALL-" + id,
		}
	}
	return domainflights.Snapshot{LastCached: lastCached, Tracks: tracks}
}

func TestGetActiveFlightsReturnsTracks(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	handler := testHandler(cacheStub{
		names:   []string{"casual"},
		flights: motionSnapshot(lastCached, "flight-bg", "flight-active"),
	})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil), "casual")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data struct {
			AvailableFilters []json.RawMessage   `json:"availableFilters"`
			Result           []ActiveFlightTrack `json:"result"`
			Meta             struct {
				RefreshIntervalMins int `json:"refreshIntervalMins"`
			} `json:"meta"`
			Pagination struct {
				TotalLength int `json:"totalLength"`
				PageLength  int `json:"pageLength"`
				PageNumber  int `json:"pageNumber"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.AvailableFilters) != 0 {
		t.Fatalf("availableFilters = %#v", body.Data.AvailableFilters)
	}
	if len(body.Data.Result) != 2 || body.Data.Pagination.TotalLength != 2 {
		t.Fatalf("body = %#v", body.Data)
	}
	if body.Data.Pagination.PageLength != domainflights.DefaultPageLength || body.Data.Pagination.PageNumber != domainflights.DefaultPageNumber {
		t.Fatalf("pagination = %#v", body.Data.Pagination)
	}
	if body.Data.Result[0].FlightID != "flight-active" || body.Data.Result[0].Speed != 401 || body.Data.Result[0].Callsign != "CALL-flight-active" {
		t.Fatalf("sorted result = %#v", body.Data.Result)
	}
}

func TestGetActiveFlightsRejectsUnknownServer(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/expert", nil), "expert")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetActiveFlightsReturnsServiceUnavailableOnCacheMiss(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}, err: cache.ErrMiss})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil), "casual")
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetActiveFlightsRejectsSnapshotWithoutTimestamp(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}, flights: domainflights.Snapshot{}})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil), "casual")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetActiveFlightsTreatsNamesReadError(t *testing.T) {
	handler := testHandler(cacheStub{namesErr: errors.New("redis down")})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil), "casual")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func decodeActivePagination(t *testing.T, recorder *httptest.ResponseRecorder) (resultCount, totalLength, pageLength, pageNumber int) {
	t.Helper()
	var body struct {
		Data struct {
			Result     []ActiveFlightTrack `json:"result"`
			Pagination struct {
				TotalLength int `json:"totalLength"`
				PageLength  int `json:"pageLength"`
				PageNumber  int `json:"pageNumber"`
			} `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return len(body.Data.Result), body.Data.Pagination.TotalLength, body.Data.Pagination.PageLength, body.Data.Pagination.PageNumber
}

func TestGetActiveFlightsDefaultsToPageSizeFifty(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	ids := make([]string, 60)
	for i := range ids {
		ids[i] = "flight-" + strconv.Itoa(i)
	}
	handler := testHandler(cacheStub{
		names:   []string{"casual"},
		flights: motionSnapshot(lastCached, ids...),
	})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual", nil), "casual")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	resultCount, totalLength, pageLength, pageNumber := decodeActivePagination(t, recorder)
	if resultCount != 50 || totalLength != 60 || pageLength != 50 || pageNumber != 1 {
		t.Fatalf("resultCount=%d totalLength=%d pageLength=%d pageNumber=%d", resultCount, totalLength, pageLength, pageNumber)
	}
}

func TestGetActiveFlightsPaginatesResults(t *testing.T) {
	lastCached := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	ids := make([]string, 60)
	for i := range ids {
		ids[i] = "flight-" + strconv.Itoa(i)
	}
	handler := testHandler(cacheStub{
		names:   []string{"casual"},
		flights: motionSnapshot(lastCached, ids...),
	})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?pageNumber=2&pageLength=10", nil), "casual")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	resultCount, totalLength, pageLength, pageNumber := decodeActivePagination(t, recorder)
	if resultCount != 10 || totalLength != 60 || pageLength != 10 || pageNumber != 2 {
		t.Fatalf("resultCount=%d totalLength=%d pageLength=%d pageNumber=%d", resultCount, totalLength, pageLength, pageNumber)
	}
}

func TestGetActiveFlightsReturnsEmptyPagePastEnd(t *testing.T) {
	handler := testHandler(cacheStub{
		names:   []string{"casual"},
		flights: motionSnapshot(time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC), "flight-1"),
	})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?pageNumber=3&pageLength=50", nil), "casual")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	resultCount, totalLength, pageLength, pageNumber := decodeActivePagination(t, recorder)
	if resultCount != 0 || totalLength != 1 || pageLength != 50 || pageNumber != 3 {
		t.Fatalf("resultCount=%d totalLength=%d pageLength=%d pageNumber=%d", resultCount, totalLength, pageLength, pageNumber)
	}
}

func TestGetActiveFlightsRejectsInvalidPage(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?pageNumber=0", nil), "casual")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetActiveFlightsRejectsInvalidPageLength(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?pageLength=0", nil), "casual")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetActiveFlightsRejectsPageLengthAboveMax(t *testing.T) {
	handler := testHandler(cacheStub{names: []string{"casual"}})
	recorder := httptest.NewRecorder()
	handler.GetActiveFlights(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/game/flights/active/casual?pageLength=5001", nil), "casual")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}
