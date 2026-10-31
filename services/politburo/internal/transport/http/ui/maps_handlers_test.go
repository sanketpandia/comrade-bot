package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/routes"
)

func TestMapActiveFlightsPartialRedirectsFullPageNavigation(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest("GET", routes.MapsFlightsActiveFlights+"?serverId=casual", nil)
	recorder := httptest.NewRecorder()
	handler.MapActiveFlightsPartial(recorder, req)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", recorder.Code)
	}
	if loc := recorder.Header().Get("Location"); loc != routes.MapsFlightsActive+"?serverId=casual" {
		t.Fatalf("Location = %q", loc)
	}
}

func TestMapActiveFlightsPartialSetsReplaceURLHeader(t *testing.T) {
	handler := newUIHandler(t, &sessionManagerStub{}, &ticketConsumerStub{})
	req := httptest.NewRequest("GET", routes.MapsFlightsActiveFlights+"?serverId=casual", nil)
	req.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()
	handler.MapActiveFlightsPartial(recorder, req)
	if got := recorder.Header().Get("HX-Replace-Url"); got != routes.MapsFlightsActive+"?serverId=casual" {
		t.Fatalf("HX-Replace-Url = %q", got)
	}
	if !strings.Contains(recorder.Body.String(), `id="maps-tracks"`) {
		t.Fatalf("expected partial body, got %s", recorder.Body.String())
	}
}
