package ui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMapsActivePageURL(t *testing.T) {
	req := httptest.NewRequest("GET", "/maps/flights/active/flights?serverId=casual&showCallsigns=1&mode=embed&header=0&pilotState=active&pilotState=away_parked&callSign=UAL", nil)
	got := mapsActivePageURL(req)
	if !strings.Contains(got, "/maps/flights/active?") {
		t.Fatalf("mapsActivePageURL() = %q", got)
	}
	for _, part := range []string{"serverId=casual", "pilotState=active", "pilotState=away_parked", "callSign=UAL", "mode=embed", "header=0"} {
		if !strings.Contains(got, part) {
			t.Fatalf("mapsActivePageURL() = %q, missing %q", got, part)
		}
	}
	if strings.Contains(got, "showCallsigns=1") {
		t.Fatalf("mapsActivePageURL() should omit default showCallsigns=1, got %q", got)
	}
}

func TestMapsActivePageURLOmitsDefaultShowCallsigns(t *testing.T) {
	req := httptest.NewRequest("GET", "/maps/flights/active/flights?serverId=casual", nil)
	got := mapsActivePageURL(req)
	if got != "/maps/flights/active?serverId=casual" {
		t.Fatalf("mapsActivePageURL() = %q", got)
	}
}

func TestMapsActivePageURLShowCallsignsOff(t *testing.T) {
	req := httptest.NewRequest("GET", "/maps/flights/active/flights?serverId=casual&showCallsigns=0", nil)
	got := mapsActivePageURL(req)
	if got != "/maps/flights/active?serverId=casual&showCallsigns=0" {
		t.Fatalf("mapsActivePageURL() = %q", got)
	}
}
