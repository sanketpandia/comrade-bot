package vaflightindex

import (
	"testing"

	gameflights "infinite-experiment/politburo/internal/livegame/flights"
)

func TestBuildVAFlightIndexMultiServerAndMultiVA(t *testing.T) {
	input := BuildInput{
		ActiveFlightIDs: map[string][]string{
			"expert": {"f1", "f2"},
			"casual": {"f3"},
		},
		Records: map[string]gameflights.Flight{
			"f1": {FlightID: "f1", MatchedVirtualAirlines: []gameflights.VirtualAirlineMatch{{VAID: "va-1"}}},
			"f2": {FlightID: "f2", MatchedVirtualAirlines: []gameflights.VirtualAirlineMatch{{VAID: "va-1"}, {VAID: "va-2"}}},
			"f3": {FlightID: "f3", MatchedVirtualAirlines: []gameflights.VirtualAirlineMatch{{VAID: "va-2"}}},
		},
	}
	index := BuildVAFlightIndex(input)
	if len(index["va-1"]) != 2 || index["va-1"][0] != "f1" || index["va-1"][1] != "f2" {
		t.Fatalf("va-1 = %v", index["va-1"])
	}
	if len(index["va-2"]) != 2 || index["va-2"][0] != "f2" || index["va-2"][1] != "f3" {
		t.Fatalf("va-2 = %v", index["va-2"])
	}
}

func TestBuildVAFlightIndexSkipsMissingRecord(t *testing.T) {
	input := BuildInput{
		ActiveFlightIDs: map[string][]string{"expert": {"f1", "f-miss"}},
		Records: map[string]gameflights.Flight{
			"f1": {FlightID: "f1", MatchedVirtualAirlines: []gameflights.VirtualAirlineMatch{{VAID: "va-1"}}},
		},
	}
	index := BuildVAFlightIndex(input)
	if len(index) != 1 || len(index["va-1"]) != 1 {
		t.Fatalf("index = %v", index)
	}
}

func TestBuildVAFlightIndexDedupesFlightPerVA(t *testing.T) {
	input := BuildInput{
		ActiveFlightIDs: map[string][]string{
			"expert": {"f1"},
			"casual": {"f1"},
		},
		Records: map[string]gameflights.Flight{
			"f1": {FlightID: "f1", MatchedVirtualAirlines: []gameflights.VirtualAirlineMatch{{VAID: "va-1"}}},
		},
	}
	index := BuildVAFlightIndex(input)
	if len(index["va-1"]) != 1 {
		t.Fatalf("va-1 = %v", index["va-1"])
	}
}
