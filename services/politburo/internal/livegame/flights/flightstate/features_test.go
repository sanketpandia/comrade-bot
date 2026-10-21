package flightstate

import (
	"math"
	"testing"
	"time"
)

func TestHaversineKnownDistance(t *testing.T) {
	// Roughly 60 NM along the equator for one degree longitude at equator.
	d := haversineNM(0, 0, 0, 1)
	if d < 59 || d > 61 {
		t.Fatalf("distance = %v", d)
	}
}

func TestPairFeaturesNonPositiveElapsed(t *testing.T) {
	cfg := DefaultConfig()
	prev := Sample{LastReportValid: true, LastReport: time.Unix(100, 0).UTC(), LatitudeValid: true, LongitudeValid: true, Latitude: 0, Longitude: 0, AltitudeValid: true, Altitude: 1000, SpeedValid: true, Speed: 100}
	cur := prev
	cur.LastReport = prev.LastReport
	p := PairFeaturesBetween(prev, cur, cfg, cur.LastReport)
	if p.ElapsedSec != 0 {
		t.Fatalf("elapsed = %v", p.ElapsedSec)
	}
}

func TestSampleMissingNeverTreatedAsZero(t *testing.T) {
	s := Sample{AltitudeValid: false, SpeedValid: false}
	if s.coreValid() {
		t.Fatal("expected invalid core")
	}
}

func TestShortestAngleTrackWrap(t *testing.T) {
	d := shortestAngleDelta(359, 1)
	if math.Abs(d-2) > 0.01 {
		t.Fatalf("delta = %v", d)
	}
}

func TestMovementToleranceJitter(t *testing.T) {
	cfg := DefaultConfig()
	prev := Sample{
		LastReportValid: true, LastReport: time.Unix(0, 0).UTC(),
		LatitudeValid: true, LongitudeValid: true, Latitude: 10, Longitude: 10,
		AltitudeValid: true, Altitude: 1000, SpeedValid: true, Speed: 10,
	}
	cur := prev
	cur.LastReport = prev.LastReport.Add(time.Minute)
	cur.Latitude += 0.00001
	p := PairFeaturesBetween(prev, cur, cfg, cur.LastReport)
	if p.Moved {
		t.Fatal("expected jitter below tolerance to be stationary")
	}
}
