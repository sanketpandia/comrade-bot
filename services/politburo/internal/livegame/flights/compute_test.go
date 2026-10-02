package flights

import (
	"testing"
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

func TestComputeFlightFastPathWhenMotionUnchanged(t *testing.T) {
	liveryID := "df597aaf-456c-4878-9d84-45201f2aae74"
	lookup := gameliveries.NewLookup(nil)
	now := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)

	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, LiveryID: liveryID,
		PilotState: PilotStateInBackground, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, AircraftName: "A350",
		PathSync: &PathSync{},
	}
	motion := MotionFromUpstream(upstream)

	out := ComputeFlight(ComputeDeps{Lookup: lookup, Metrics: metrics.NewRegistry()},
		infiniteflight.Session{NormalizedName: "casual"}, upstream, prior, motion, true, now)
	if !out.Decision.FastPath || out.Decision.UpdateFlightRecord {
		t.Fatalf("decision = %#v", out.Decision)
	}
	if out.Flight.AircraftName != "A350" {
		t.Fatalf("flight = %#v", out.Flight)
	}
}

func TestComputeFlightFPLForcesFullPathDespiteUnchangedMotion(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Speed: 100, PilotState: PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &Flight{
		FlightID: "f1", Speed: 100,
		PathSync: &PathSync{LastFPLSyncAt: now.Add(-FPLSyncInterval)},
	}
	motion := MotionFromUpstream(upstream)

	out := ComputeFlight(ComputeDeps{Lookup: gameliveries.NewLookup(nil), Metrics: metrics.NewRegistry()},
		infiniteflight.Session{NormalizedName: "casual"}, upstream, prior, motion, true, now)
	if out.Decision.FastPath {
		t.Fatal("expected full compute for FPL")
	}
	if !out.Decision.FPLSync || !out.Decision.UpdateFlightRecord {
		t.Fatalf("decision = %#v", out.Decision)
	}
	if out.Flight.PathSync == nil || !out.Flight.PathSync.FPLSyncRequired {
		t.Fatalf("pathSync = %#v", out.Flight.PathSync)
	}
}
