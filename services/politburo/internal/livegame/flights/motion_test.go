package flights

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func TestMotionFromUpstreamMatchesMapFlightRounding(t *testing.T) {
	motion := MotionFromUpstream(infiniteflight.Flight{
		Latitude:  47.451234,
		Longitude: 8.561234,
		Speed:     525.6,
	})
	if motion.Latitude != 47.4512 || motion.Longitude != 8.5612 || motion.Speed != 526 {
		t.Fatalf("motion = %#v", motion)
	}
}

func TestMotionFromUpstreamIncludesCallsign(t *testing.T) {
	motion := MotionFromUpstream(infiniteflight.Flight{Callsign: "SWA123"})
	if motion.Callsign != "SWA123" {
		t.Fatalf("motion = %#v", motion)
	}
}

func TestMotionEqual(t *testing.T) {
	a := FlightMotion{Latitude: 1, Longitude: 2, Speed: 3}
	b := FlightMotion{Latitude: 1, Longitude: 2, Speed: 3}
	if !MotionEqual(a, b) {
		t.Fatal("expected equal motion")
	}
	b.Speed = 4
	if MotionEqual(a, b) {
		t.Fatal("expected unequal motion")
	}
}

func TestSortedTrackFlightIDs(t *testing.T) {
	ids := SortedTrackFlightIDs(map[string]FlightMotion{"b": {}, "a": {}, "c": {}})
	if len(ids) != 3 || ids[0] != "a" || ids[1] != "b" || ids[2] != "c" {
		t.Fatalf("ids = %#v", ids)
	}
}
