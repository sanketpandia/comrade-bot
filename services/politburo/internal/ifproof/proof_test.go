package ifproof

import (
	"testing"

	"infinite-experiment/politburo/internal/infiniteflight"
)

func TestParseRouteProof(t *testing.T) {
	origin, dest, err := ParseRouteProof("egll-ksea")
	if err != nil {
		t.Fatalf("ParseRouteProof() error = %v", err)
	}
	if origin != "EGLL" || dest != "KSEA" {
		t.Fatalf("route = %s-%s", origin, dest)
	}
}

func TestParseRouteProofRejectsBadShape(t *testing.T) {
	_, _, err := ParseRouteProof("EGLL")
	if err == nil {
		t.Fatal("expected error for single segment")
	}
}

func TestMatchesRecentFlights(t *testing.T) {
	flights := []infiniteflight.LogbookFlight{
		{Origin: "YTYA", Destination: "YSSY"},
		{Origin: "EGLL", Destination: "KSEA"},
	}
	if !MatchesRecentFlights(flights, "EGLL", "KSEA") {
		t.Fatal("expected match on second flight")
	}
	if MatchesRecentFlights(flights, "KSEA", "EGLL") {
		t.Fatal("reverse route should not match")
	}
}
