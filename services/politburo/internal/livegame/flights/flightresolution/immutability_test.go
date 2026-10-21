package flightresolution

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights"
	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
	"infinite-experiment/politburo/internal/livegame/infiniteflight"
	gameliveries "infinite-experiment/politburo/internal/livegame/liveries"
	"infinite-experiment/politburo/internal/metrics"
)

func TestEffectsDoNotMutateRawOrPrior(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	upstream := infiniteflight.Flight{
		FlightID: "f1", Callsign: "IE123", Speed: 120, LiveryID: "l1", AircraftID: "a1",
		PilotState: flights.PilotStateActive, LastReport: "2026-08-15 05:09:53Z",
	}
	raw := flights.NewObservation(upstream, infiniteflight.Session{ID: "s1", NormalizedName: "expert"})
	prior := &flights.Record{
		Raw: flights.Observation{FlightID: "f1", Callsign: "IE123", Speed: 100, LiveryID: "l1", AircraftID: "a1"},
		Derived: flights.Derived{
			AircraftName: "A320",
			PathSync:     flights.PathSync{LastFPLSyncAt: now.Add(-flights.FPLSyncInterval)},
		},
	}
	rawBefore, _ := json.Marshal(raw)
	priorBefore, _ := json.Marshal(prior)

	ctx := ComputeContext{
		Context:        context.Background(),
		Deps:           ComputeDeps{Lookup: gameliveries.NewLookup(nil), Metrics: metrics.NewRegistry()},
		Session:        infiniteflight.Session{NormalizedName: "expert"},
		RefreshedAt:    now,
		PriorMotion:    flights.MotionFromUpstream(upstream),
		HasPriorMotion: true,
		LiveVACatalog:  expertIECatalog(),
		History:        flightstate.History{},
	}
	facts := computeFacts(ctx, raw, prior)
	in := EffectInput{Ctx: ctx, Raw: raw, Prior: prior, Facts: facts, History: ctx.History}
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
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
