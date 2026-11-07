package flightresolution

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/flights"
)

func TestLiveryResolveMetricAppliesOnIdentityChange(t *testing.T) {
	prior := &flights.Record{
		Raw: flights.Observation{AircraftID: "a1", LiveryID: "l1"},
	}
	raw := flights.Observation{AircraftID: "a1", LiveryID: "l1"}
	in := EffectInput{
		Raw:   raw,
		Prior: prior,
		Facts: computeFacts(ComputeContext{}, raw, prior),
	}
	if liveryResolveMetricApplies(in) {
		t.Fatal("expected skip when identity unchanged")
	}

	raw = flights.Observation{AircraftID: "a2", LiveryID: "l1"}
	in = EffectInput{
		Raw:   raw,
		Prior: prior,
		Facts: computeFacts(ComputeContext{}, raw, prior),
	}
	if !liveryResolveMetricApplies(in) {
		t.Fatal("expected metric when aircraft changed")
	}

	in = EffectInput{
		Raw:   flights.Observation{AircraftID: "a1", LiveryID: "l1"},
		Facts: computeFacts(ComputeContext{}, flights.Observation{AircraftID: "a1", LiveryID: "l1"}, nil),
	}
	if !liveryResolveMetricApplies(in) {
		t.Fatal("expected metric without prior")
	}
}
