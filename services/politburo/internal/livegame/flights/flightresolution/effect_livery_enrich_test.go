package flightresolution

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestLiveryResolveNeeded(t *testing.T) {
	prior := &flights.Flight{AircraftID: "a1", LiveryID: "l1", AircraftName: "A320"}
	if liveryResolveNeeded(prior, "a1", "l1") {
		t.Fatal("expected skip when names already resolved")
	}
	if !liveryResolveNeeded(prior, "a2", "l1") {
		t.Fatal("expected resolve when aircraft changed")
	}
	if !liveryResolveNeeded(nil, "a1", "l1") {
		t.Fatal("expected resolve without prior")
	}
}
