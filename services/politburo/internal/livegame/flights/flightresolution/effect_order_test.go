package flightresolution

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestFlightEffectsOrder(t *testing.T) {
	want := []string{
		"FlightUsers",
		"VAMatch",
		"LiveModeCheck",
		"FlightState",
		"PositionTrail",
		"FPLSyncUpdate",
	}
	if len(flightEffects) != len(want) {
		t.Fatalf("len(flightEffects) = %d, want %d", len(flightEffects), len(want))
	}
	for i, name := range want {
		if flightEffects[i].Name != name {
			t.Fatalf("flightEffects[%d].Name = %q, want %q", i, flightEffects[i].Name, name)
		}
	}
}

func TestApplyLiveModeCheckNoOp(t *testing.T) {
	analytics := flights.Analytics{}
	actions := flights.Actions{FPLSync: flights.FPLSyncAction{Required: true}}
	links := flights.Links{Users: []string{"u1"}}
	outcome := Outcome{FPLSyncScheduled: true}

	applyLiveModeCheck(EffectInput{}, &analytics, &actions, &links, &outcome)

	if analytics.FlightState != nil {
		t.Fatal("analytics mutated")
	}
	if !actions.FPLSync.Required {
		t.Fatal("actions mutated")
	}
	if len(links.Users) != 1 || links.Users[0] != "u1" {
		t.Fatalf("links mutated: %#v", links)
	}
	if !outcome.FPLSyncScheduled {
		t.Fatal("outcome mutated")
	}
}
