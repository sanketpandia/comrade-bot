package proof

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
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

func TestMatchesLatestCompleteFlight(t *testing.T) {
	flights := []infiniteflight.LogbookFlight{
		{Origin: "EGLL", Destination: ""},
		{Origin: "", Destination: "KSEA"},
		{Origin: "YTYA", Destination: "YSSY"},
		{Origin: "EGLL", Destination: "KSEA"},
	}
	if !MatchesLatestCompleteFlight(flights, "YTYA", "YSSY") {
		t.Fatal("expected match on latest complete flight YTYA-YSSY")
	}
	if MatchesLatestCompleteFlight(flights, "EGLL", "KSEA") {
		t.Fatal("older complete row must not match when a newer complete row exists")
	}
	if MatchesLatestCompleteFlight(flights, "KSEA", "EGLL") {
		t.Fatal("reverse route should not match")
	}
}

func TestMatchesLatestCompleteFlightNoCompleteRow(t *testing.T) {
	flights := []infiniteflight.LogbookFlight{
		{Origin: "EGLL", Destination: ""},
		{Origin: "", Destination: "KSEA"},
	}
	if MatchesLatestCompleteFlight(flights, "EGLL", "KSEA") {
		t.Fatal("expected no match when no complete row exists")
	}
}
