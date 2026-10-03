package flightresolution

import (
	"context"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

func testComputeContext(session infiniteflight.Session, motion flights.FlightMotion, hasMotion bool, at time.Time) ComputeContext {
	return ComputeContext{
		Context:        context.Background(),
		Deps:           ComputeDeps{Lookup: gameliveries.NewLookup(nil), Metrics: metrics.NewRegistry()},
		Session:        session,
		RefreshedAt:    at,
		PriorMotion:    motion,
		HasPriorMotion: hasMotion,
	}
}

func TestResolveFlightSkipsEffectsWhenMotionUnchanged(t *testing.T) {
	liveryID := "df597aaf-456c-4878-9d84-45201f2aae74"
	lookup := gameliveries.NewLookup(nil)
	now := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)

	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, LiveryID: liveryID,
		PilotState: flights.PilotStateInBackground, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &flights.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, AircraftName: "A350",
		PathSync: &flights.PathSync{},
	}
	motion := flights.MotionFromUpstream(upstream)

	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)
	ctx.Deps.Lookup = lookup

	out := ResolveFlight(ctx, upstream, prior)
	if out.Outcome.UpdateFlightRecord || out.Outcome.FPLSyncScheduled {
		t.Fatalf("outcome = %#v", out.Outcome)
	}
	if out.Flight.AircraftName != "A350" {
		t.Fatalf("flight = %#v", out.Flight)
	}
}

func TestResolveFlightFPLSchedulesSyncDespiteUnchangedMotion(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Speed: 100, PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &flights.Flight{
		FlightID: "f1", Speed: 100,
		PathSync: &flights.PathSync{LastFPLSyncAt: now.Add(-flights.FPLSyncInterval)},
	}
	motion := flights.MotionFromUpstream(upstream)

	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)
	out := ResolveFlight(ctx, upstream, prior)
	if !out.Outcome.UpdateFlightRecord {
		t.Fatal("expected record update for FPL tick")
	}
	if !out.Outcome.FPLSyncScheduled {
		t.Fatalf("outcome = %#v", out.Outcome)
	}
	if out.Flight.PathSync == nil || !out.Flight.PathSync.FPLSyncRequired {
		t.Fatalf("pathSync = %#v", out.Flight.PathSync)
	}
}
