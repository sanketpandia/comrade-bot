package ui

import (
	"net/http/httptest"
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/routes"
)

func TestMapsActivePageURL(t *testing.T) {
	req := httptest.NewRequest("GET", routes.MapsFlightsActiveFlights+"?serverId=casual&showCallsigns=1&mode=embed&header=0&pilotState=active&pilotState=away_parked&callSign=UAL", nil)
	got := mapsActivePageURL(req)
	if !strings.Contains(got, routes.MapsFlightsActive+"?") {
		t.Fatalf("mapsActivePageURL() = %q", got)
	}
	for _, part := range []string{"serverId=casual", "pilotState=active", "pilotState=away_parked", "callSign=UAL", "mode=embed", "header=0"} {
		if !strings.Contains(got, part) {
			t.Fatalf("mapsActivePageURL() = %q, missing %q", got, part)
		}
	}
}

func TestMapsActivePageURLMinimal(t *testing.T) {
	req := httptest.NewRequest("GET", routes.MapsFlightsActiveFlights+"?serverId=casual", nil)
	got := mapsActivePageURL(req)
	if got != routes.MapsFlightsActive+"?serverId=casual" {
		t.Fatalf("mapsActivePageURL() = %q", got)
	}
}

func TestMapsActivePageURLShowCallsignsZero(t *testing.T) {
	req := httptest.NewRequest("GET", routes.MapsFlightsActiveFlights+"?serverId=casual&showCallsigns=0", nil)
	got := mapsActivePageURL(req)
	if got != routes.MapsFlightsActive+"?serverId=casual&showCallsigns=0" {
		t.Fatalf("mapsActivePageURL() = %q", got)
	}
}
