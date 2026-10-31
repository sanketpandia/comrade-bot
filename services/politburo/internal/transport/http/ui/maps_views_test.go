package ui

import (
	"testing"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	domainflights "infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

func TestRecordViewAnalyticsAndDebugKeys(t *testing.T) {
	since := time.Date(2026, 10, 31, 20, 15, 0, 0, time.UTC)
	record := domainflights.Record{
		Raw: domainflights.Observation{
			FlightID:  "flight-1",
			UserID:    "user-1",
			SessionID: "session-1",
			Latitude:  52.3105,
			Longitude: 4.7683,
		},
		Derived: domainflights.Derived{
			Users:                  []string{"user-0", "user-1"},
			MatchedVirtualAirlines: []virtualairlines.VirtualAirlineMatch{{VAID: "va-1", VACode: "TEST"}},
			FlightState: &flightstate.Summary{
				Phase:         flightstate.PhaseAwaitingAPPlus,
				PreviousPhase: flightstate.PhaseCruise,
				Activity:      flightstate.ActivityMoving,
				Confidence:    0.5,
				PhaseSince:    since,
				APPlus: flightstate.APPlus{
					Candidate: true,
					Score:     0.95,
					Since:     since,
					Reasons:   []string{"pilot_state_in_background"},
				},
			},
		},
	}

	view := recordViewFromDomain(record)

	if view.Latitude != "52.3105" || view.Longitude != "4.7683" {
		t.Fatalf("coordinates = %q, %q", view.Latitude, view.Longitude)
	}
	a := view.Analytics
	if a == nil {
		t.Fatal("missing analytics")
	}
	if a.Phase != "Awaiting AP+ resumption" || a.PreviousPhase != "Cruise" || a.Activity != "Moving" {
		t.Fatalf("phase labels = %q / %q / %q", a.Phase, a.PreviousPhase, a.Activity)
	}
	if a.ConfidenceLevel != "medium" || a.PhaseSince != "20:15:00 UTC" {
		t.Fatalf("confidence %q, since %q", a.ConfidenceLevel, a.PhaseSince)
	}
	if !a.APPlusCandidate || a.APPlusScore != "95%" || len(a.APPlusReasons) != 1 || a.APPlusReasons[0] != "Pilot state in background" {
		t.Fatalf("ap+ = %+v", a)
	}

	sources := map[string]string{}
	for _, key := range view.DebugKeys {
		sources[key.Value] = key.Source
	}
	for value, want := range map[string]string{"flight-1": "IF", "user-1": "IF", "session-1": "IF", "user-0": "IF", "va-1": "CB"} {
		if sources[value] != want {
			t.Fatalf("debug key %q source = %q, want %q (keys %+v)", value, sources[value], want, view.DebugKeys)
		}
	}
}
