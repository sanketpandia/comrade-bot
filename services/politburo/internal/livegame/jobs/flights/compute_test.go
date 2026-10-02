package flights

import (
	"context"
	"testing"
	"time"

	gameflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

func TestComputeFlightFastPathWhenMotionUnchanged(t *testing.T) {
	liveryID := "df597aaf-456c-4878-9d84-45201f2aae74"
	job := New(flightsClientStub{}, nil, testLookup(liveryID, "A350", "Swiss"), metrics.NewRegistry())
	now := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)
	job.now = func() time.Time { return now }

	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, LiveryID: liveryID,
		PilotState: gameflights.PilotStateInBackground, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &gameflights.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, AircraftName: "A350",
		PathSync: &gameflights.PathSync{},
	}
	motion := gameflights.MotionFromUpstream(upstream)

	out := job.computeFlight(context.Background(), infiniteflight.Session{NormalizedName: "casual"}, upstream, prior, motion, true, now)
	if !out.Decision.FastPath || out.Decision.UpdateFlightRecord {
		t.Fatalf("decision = %#v", out.Decision)
	}
	if out.Flight.AircraftName != "A350" {
		t.Fatalf("flight = %#v", out.Flight)
	}
}

func TestComputeFlightFPLForcesFullPathDespiteUnchangedMotion(t *testing.T) {
	job := New(flightsClientStub{}, nil, gameliveries.NewLookup(nil), metrics.NewRegistry())
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Speed: 100, PilotState: gameflights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &gameflights.Flight{
		FlightID: "f1", Speed: 100,
		PathSync: &gameflights.PathSync{LastFPLSyncAt: now.Add(-gameflights.FPLSyncInterval)},
	}
	motion := gameflights.MotionFromUpstream(upstream)

	out := job.computeFlight(context.Background(), infiniteflight.Session{NormalizedName: "casual"}, upstream, prior, motion, true, now)
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
