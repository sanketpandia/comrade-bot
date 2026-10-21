package flightresolution

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/community/virtualairlines"
	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

func TestEffectsDoNotMutateRawOrPrior(t *testing.T) {
	runImmutabilityCase(t, false)
	runImmutabilityCase(t, true)
}

func runImmutabilityCase(t *testing.T, motionChanged bool) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 120, LiveryID: "l1", AircraftID: "a1",
		PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	if motionChanged {
		upstream.Speed = 140
	}
	raw := flights.NewObservation(upstream, infiniteflight.Session{ID: "s1", NormalizedName: "expert"}, now)
	flightSummary := &flightstate.Summary{
		Phase: flightstate.PhaseCruise, Activity: flightstate.ActivityMoving,
		DataQuality: flightstate.DataQualityFresh, Confidence: 0.7, PhaseSince: now,
		Reasons: []string{"level_progress"},
		APPlus: flightstate.APPlus{Candidate: true, Score: 0.6, Since: now, Reasons: []string{"stationary_high_alt"}},
	}
	prior := &flights.Record{
		Raw: flights.Observation{FlightID: "f1", Callsign: "IE123", Speed: 100, LiveryID: "l1", AircraftID: "a1"},
		Derived: flights.Derived{
			AircraftName: "A320",
			PathSync:     flights.PathSync{LastFPLSyncAt: now.Add(-flights.FPLSyncInterval)},
			MatchedVirtualAirlines: []virtualairlines.VirtualAirlineMatch{{VAID: "va-1", VACode: "IE"}},
			FlightState:            flightSummary,
		},
	}
	history := flightstate.History{}
	history, _ = history.Push(flights.FlightStateSample(raw), flightstate.DefaultConfig())

	rawBefore, _ := json.Marshal(raw)
	priorBefore, _ := json.Marshal(prior)
	historyBefore, _ := json.Marshal(history)

	ctx := ComputeContext{
		Context:        context.Background(),
		Deps:           ComputeDeps{Lookup: gameliveries.NewLookup(nil), Metrics: metrics.NewRegistry()},
		Session:        infiniteflight.Session{NormalizedName: "expert"},
		RefreshedAt:    now,
		PriorMotion:    flights.MotionFromUpstream(upstream),
		HasPriorMotion: true,
		LiveVACatalog:  expertIECatalog(),
	}
	facts := computeFacts(ctx, raw, prior)
	if motionChanged {
		facts.MotionChanged = true
	}
	in := EffectInput{Ctx: ctx, Raw: raw, Prior: prior, Facts: facts, History: history}
	derived := prior.Derived.Clone()
	outcome := Outcome{}

	for _, effect := range flightEffects {
		if effect.Applies(in) {
			effect.Apply(in, &derived, &outcome)
		}
	}

	if !bytes.Equal(rawBefore, mustJSON(raw)) {
		t.Fatal("raw observation mutated during effect application")
	}
	if !bytes.Equal(priorBefore, mustJSON(prior)) {
		t.Fatal("prior record mutated during effect application")
	}
	if !bytes.Equal(historyBefore, mustJSON(history)) {
		t.Fatal("input history mutated during effect application")
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
