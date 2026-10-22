package flightstate

import (
	"testing"
	"time"
)

func TestSustainedClimbCountTrailingOnly(t *testing.T) {
	cfg := DefaultConfig()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	interval := time.Minute

	// Early climb pairs, then level flight on the most recent pairs.
	climbStep := 500.0 // ft/min over 1 min between samples
	levelAlt := 34000.0
	samples := []Sample{
		sampleAt(t0, 10000, 250, climbStep),
		sampleAt(t0.Add(interval), 15000, 280, climbStep),
		sampleAt(t0.Add(2*interval), 20000, 300, climbStep),
		sampleAt(t0.Add(3*interval), 25000, 320, climbStep),
		sampleAt(t0.Add(4*interval), 30000, 350, climbStep),
		sampleAt(t0.Add(5*interval), levelAlt, 393, 0),
		sampleAt(t0.Add(6*interval), levelAlt, 393, 0),
	}
	h := seq(samples...)
	w := WindowFeaturesFrom(h, cfg, t0.Add(6*interval))

	if w.SustainedClimbCount != 0 {
		t.Fatalf("expected sustained climb 0 after level pairs, got %d", w.SustainedClimbCount)
	}
	if w.Pair.AltStable && w.SustainedClimbCount > 0 {
		t.Fatal("stable level pair should not leave a positive sustained climb count")
	}
}

func TestSustainedClimbCountActiveClimb(t *testing.T) {
	cfg := DefaultConfig()
	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	interval := time.Minute
	climbStep := 500.0

	samples := []Sample{
		sampleAt(t0, 10000, 250, climbStep),
		sampleAt(t0.Add(interval), 15000, 280, climbStep),
		sampleAt(t0.Add(2*interval), 20000, 300, climbStep),
	}
	h := seq(samples...)
	w := WindowFeaturesFrom(h, cfg, t0.Add(2*interval))

	if w.SustainedClimbCount != 2 {
		t.Fatalf("expected sustained climb 2 for two trailing climb pairs, got %d", w.SustainedClimbCount)
	}
}
