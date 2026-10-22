package flightstate

import (
	"testing"
	"time"
)

func seq(samples ...Sample) History {
	cfg := DefaultConfig()
	h := History{}
	for _, s := range samples {
		h, _ = h.Push(s, cfg)
	}
	return h
}

func sampleAt(t time.Time, alt, speed, vs float64) Sample {
	return Sample{
		LastReport: t, LastReportValid: true, ObservedAt: t,
		LatitudeValid: true, LongitudeValid: true, Latitude: 40.0, Longitude: -75.0,
		AltitudeValid: true, Altitude: alt,
		SpeedValid: true, Speed: speed,
		VerticalSpeedValid: true, VerticalSpeed: vs,
		TrackValid: true, Track: 90,
		IsConnected: true,
	}
}

func TestClassifySpawnedInFirstSample(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	h := seq(sampleAt(t0, 100, 0, 0))
	st := Classify(h.State, h, DefaultConfig(), t0)
	if st.Phase != PhaseSpawnedIn {
		t.Fatalf("phase = %s", st.Phase)
	}
}

func TestScoreParkedOnGround(t *testing.T) {
	cfg := DefaultConfig()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s1 := sampleAt(t0, 50, 0, 0)
	s2 := sampleAt(t0.Add(time.Minute), 50, 0, 0)
	s3 := sampleAt(t0.Add(2*time.Minute), 50, 0, 0)
	h := seq(s1, s2, s3)
	prev := State{Phase: PhaseTaxiing, GroundRefAltFt: 50, PhaseSince: t0}
	w := WindowFeaturesFrom(h, cfg, t0.Add(time.Minute))
	c := scoreParked(prev, w, cfg)
	if c.strength < 0.5 {
		t.Fatalf("expected parked match, strength=%v", c.strength)
	}
}

func TestInBackgroundPilotStateAPPlus(t *testing.T) {
	cfg := DefaultConfig()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := sampleAt(t0, 34000, 393, 0)
	s.IsConnected = false
	s.PilotState = PilotStateInBackground
	h := seq(s)
	prev := State{Phase: PhaseClimbing, PhaseSince: t0, Confidence: 0.7}
	st := Classify(prev, h, cfg, t0)
	if !st.APPlus.Candidate {
		t.Fatal("expected AP+ candidate when pilot is in background")
	}
	if st.APPlus.Score < 0.9 {
		t.Fatalf("expected high AP+ score, got %v", st.APPlus.Score)
	}
}

func TestStaleReportNeverAPPlus(t *testing.T) {
	cfg := DefaultConfig()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := sampleAt(t0, 12000, 0, 0)
	s.IsConnected = false
	h := seq(s)
	prev := State{Phase: PhaseCruise, PhaseSince: t0, Confidence: 0.7}
	st := Classify(prev, h, cfg, t0.Add(10*time.Minute))
	if st.APPlus.Candidate {
		t.Fatal("disconnected flight should not be AP+ candidate")
	}
}
