package flightresolution

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestLiveryEnrichApplies(t *testing.T) {
	prior := &flights.Record{
		Raw:     flights.Observation{AircraftID: "a1", LiveryID: "l1"},
		Derived: flights.Derived{AircraftName: "A320"},
	}
	raw := flights.Observation{AircraftID: "a1", LiveryID: "l1"}
	in := EffectInput{
		Raw:   raw,
		Prior: prior,
		Facts: computeFacts(ComputeContext{}, raw, prior),
	}
	if liveryEnrichApplies(in) {
		t.Fatal("expected skip when names already resolved")
	}

	raw = flights.Observation{AircraftID: "a2", LiveryID: "l1"}
	in = EffectInput{
		Raw:   raw,
		Prior: prior,
		Facts: computeFacts(ComputeContext{}, raw, prior),
	}
	if !liveryEnrichApplies(in) {
		t.Fatal("expected resolve when aircraft changed")
	}

	in = EffectInput{
		Raw:   flights.Observation{AircraftID: "a1", LiveryID: "l1"},
		Facts: computeFacts(ComputeContext{}, flights.Observation{AircraftID: "a1", LiveryID: "l1"}, nil),
	}
	if !liveryEnrichApplies(in) {
		t.Fatal("expected resolve without prior")
	}
}
