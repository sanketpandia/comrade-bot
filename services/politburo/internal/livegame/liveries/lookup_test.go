package liveries

import (
	"testing"
	"time"
)

func TestResolvePrefersLiveryMatch(t *testing.T) {
	row := AircraftLivery{
		LiveryID: "livery-1", AircraftID: "aircraft-1",
		DisplayAircraftName: "A320", DisplayLiveryName: "BA",
	}
	l := NewStaticLookup(map[string]AircraftLivery{"livery-1": row}, map[string]AircraftLivery{"aircraft-1": row})

	names, outcome := l.Resolve("livery-1", "aircraft-1")
	if outcome != MatchLivery || names.AircraftName != "A320" || names.LiveryName != "BA" {
		t.Fatalf("resolve = %#v %s", names, outcome)
	}
}

func TestResolveFallsBackToAircraft(t *testing.T) {
	l := NewStaticLookup(nil, map[string]AircraftLivery{
		"aircraft-1": {
			AircraftID: "aircraft-1", DisplayAircraftName: "B737", UpdatedAt: time.Now().UTC(),
		},
	})

	names, outcome := l.Resolve("missing-livery", "aircraft-1")
	if outcome != MatchAircraftOnly || names.AircraftName != "B737" || names.LiveryName != UnrecognizedLiveryName {
		t.Fatalf("resolve = %#v %s", names, outcome)
	}
}

func TestResolveMiss(t *testing.T) {
	l := NewLookup(nil)
	_, outcome := l.Resolve("a", "b")
	if outcome != MatchMiss {
		t.Fatalf("outcome = %s", outcome)
	}
}
