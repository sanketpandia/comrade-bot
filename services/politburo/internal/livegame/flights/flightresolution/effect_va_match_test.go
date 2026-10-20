package flightresolution

import (
	"testing"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func expertIECatalog() virtualairlines.LiveVACatalog {
	return virtualairlines.LiveVACatalog{
		Rules: []virtualairlines.CallsignAffix{
			{VAID: "va-1", VACode: "IE", Prefix: "IE", PreferredServers: []string{"expert"}},
		},
		Servers: map[string][]string{"expert": {"va-1"}},
	}
}

func TestResolveFlightVAMatchOnExpertSession(t *testing.T) {
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateInBackground,
		LastReport: "2026-08-15 05:09:53Z",
	}
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "expert"}, flights.MotionFromUpstream(upstream), false, now)
	ctx.LiveVACatalog = expertIECatalog()

	out := ResolveFlight(ctx, upstream, nil)
	if len(out.Flight.MatchedVirtualAirlines) != 1 || out.Flight.MatchedVirtualAirlines[0].VAID != "va-1" {
		t.Fatalf("flight = %#v", out.Flight)
	}
}

func TestResolveFlightVANoMatchOnWrongSession(t *testing.T) {
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 100, PilotState: flights.PilotStateInBackground,
		LastReport: "2026-08-15 05:09:53Z",
	}
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "casual"}, flights.MotionFromUpstream(upstream), false, now)
	ctx.LiveVACatalog = expertIECatalog()

	out := ResolveFlight(ctx, upstream, nil)
	if len(out.Flight.MatchedVirtualAirlines) != 0 {
		t.Fatalf("flight = %#v", out.Flight)
	}
}

func TestResolveFlightVAMatchUpdatesWhenCallsignChangesDespiteUnchangedMotion(t *testing.T) {
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE999", Speed: 100, PilotState: flights.PilotStateInBackground,
		LastReport: "2026-08-15 05:09:53Z",
	}
	prior := &flights.Flight{
		FlightID: "f1", Callsign: "OTHER", Speed: 100,
		MatchedVirtualAirlines: nil,
		PathSync:               &flights.PathSync{},
	}
	motion := flights.MotionFromUpstream(upstream)
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	ctx := testComputeContext(infiniteflight.Session{NormalizedName: "expert"}, motion, true, now)
	ctx.LiveVACatalog = expertIECatalog()

	out := ResolveFlight(ctx, upstream, prior)
	if !out.Outcome.UpdateFlightRecord {
		t.Fatal("expected record update when VA matches change")
	}
	if len(out.Flight.MatchedVirtualAirlines) != 1 {
		t.Fatalf("flight = %#v", out.Flight)
	}
}
