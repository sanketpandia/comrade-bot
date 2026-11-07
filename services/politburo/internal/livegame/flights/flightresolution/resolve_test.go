package flightresolution

import (
	"context"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
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

func TestResolveFlightSkipsRecordWriteWhenMotionUnchanged(t *testing.T) {
	liveryID := "df597aaf-456c-4878-9d84-45201f2aae74"
	now := time.Date(2026, time.August, 15, 6, 0, 0, 0, time.UTC)

	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "Swiss", Speed: 100, LiveryID: liveryID,
		PilotState: flights.PilotStateInBackground, LastReport: "2026-08-15 05:09:53Z",
	}
	motion := flights.MotionFromUpstream(upstream)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)

	warm := ResolveFlight(ctx, upstream, nil, flightstate.History{})
	prior := &flights.Record{
		Raw:       warm.Record.Raw,
		Analytics: warm.Record.Analytics.Clone(),
		Actions:   warm.Record.Actions.Clone(),
		Links:     warm.Record.Links.Clone(),
		History:   warm.NextHistory,
	}

	out := ResolveFlight(ctx, upstream, prior, warm.NextHistory)
	if out.Outcome.UpdateFlightRecord || out.Outcome.FPLSyncScheduled {
		t.Fatalf("outcome = %#v", out.Outcome)
	}
}

func TestResolveFlightFPLSchedulesSyncDespiteUnchangedMotion(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &flights.Record{
		Raw: flights.Observation{FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateActive},
		Links: flights.Links{
			VirtualAirlines: []virtualairlines.VirtualAirlineMatch{{VAID: "va-1", VACode: "IE"}},
		},
	}
	motion := flights.MotionFromUpstream(upstream)

	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)
	ctx.LiveVACatalog = virtualairlines.LiveVACatalog{
		Rules:         []virtualairlines.CallsignAffix{{VAID: "va-1", VACode: "IE", Prefix: "IE"}},
		UnscopedVAIDs: []string{"va-1"},
	}
	out := ResolveFlight(ctx, upstream, prior, flightstate.History{})
	if !out.Outcome.UpdateFlightRecord {
		t.Fatal("expected record update for FPL tick")
	}
	if !out.Outcome.FPLSyncScheduled {
		t.Fatalf("outcome = %#v", out.Outcome)
	}
	if !out.Record.Actions.FPLSync.Required {
		t.Fatalf("fplSync = %#v", out.Record.Actions.FPLSync)
	}
}

func TestResolveFlightUpdatesRecordWhenOnlyUserIDChanges(t *testing.T) {
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	userB := "0d85b360-92b5-4e62-ac64-41bd1c829772"
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", UserID: userB, Speed: 100, PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	motion := flights.MotionFromUpstream(upstream)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)

	prior := &flights.Record{
		Raw: flights.Observation{
			FlightID: "f1", UserID: userA, Speed: 100, PilotState: flights.PilotStateActive,
			LastReport: now, LastReportValid: true,
		},
		Links: flights.Links{Users: []string{userA}},
	}

	out := ResolveFlight(ctx, upstream, prior, flightstate.History{})
	if !out.Outcome.UpdateFlightRecord {
		t.Fatal("expected record update when userId changes")
	}
	if len(out.Record.Links.Users) != 2 || out.Record.Links.Users[1] != userB {
		t.Fatalf("users = %#v", out.Record.Links.Users)
	}
}

func TestResolveFlightStoresRouteEndpointsFromCachedPlan(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	motion := flights.MotionFromUpstream(upstream)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)
	ctx.PlanLoaded = true
	ctx.PlanFetchedAt = now
	ctx.Plan = flights.CachedFlightPlan{
		FetchedAt: now,
		Waypoints: []flights.PlanWaypoint{
			{Name: "EGLL"},
			{Name: "KJFK"},
		},
	}

	prior := &flights.Record{
		Raw: flights.Observation{FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateActive},
	}

	out := ResolveFlight(ctx, upstream, prior, flightstate.History{})
	if out.Record.Normalized.Departure != "EGLL" || out.Record.Normalized.Destination != "KJFK" {
		t.Fatalf("normalized route = %#v", out.Record.Normalized)
	}
	if !out.Outcome.UpdateFlightRecord {
		t.Fatal("expected record write when route endpoints change")
	}
}

func TestResolveFlightFPLSyncRequiredResetsAfterSync(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	motion := flights.MotionFromUpstream(upstream)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, motion, true, now)

	duePrior := &flights.Record{
		Raw: flights.Observation{FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateActive, ObservedAt: now},
		Links: flights.Links{
			VirtualAirlines: []virtualairlines.VirtualAirlineMatch{{VAID: "va-1", VACode: "IE"}},
		},
		Actions: flights.Actions{
			FPLSync: flights.FPLSyncAction{Required: true, LastAt: now},
		},
	}
	ctx.PlanFetchedAt = now
	ctx.LiveVACatalog = virtualairlines.LiveVACatalog{
		Rules:         []virtualairlines.CallsignAffix{{VAID: "va-1", VACode: "IE", Prefix: "IE"}},
		UnscopedVAIDs: []string{"va-1"},
	}
	out := ResolveFlight(ctx, upstream, duePrior, flightstate.History{})
	if out.Record.Actions.FPLSync.Required {
		t.Fatalf("expected fplSync.required reset when not due, got %#v", out.Record.Actions.FPLSync)
	}
}
