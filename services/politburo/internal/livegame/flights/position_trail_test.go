package flights

import (
	"testing"
	"time"
)

func TestCompactPositionTrailKeepsCorner(t *testing.T) {
	base := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	fixes := []PositionFix{
		{At: base, Latitude: 0, Longitude: 0},
		{At: base.Add(time.Minute), Latitude: 0, Longitude: 1},
		{At: base.Add(2 * time.Minute), Latitude: 1, Longitude: 1},
	}
	out := compactPositionTrail(fixes, 3)
	if len(out) != 3 {
		t.Fatalf("len = %d", len(out))
	}
}

func TestCompactPositionTrailBound(t *testing.T) {
	fixes := make([]PositionFix, PositionTrailMax+1)
	for i := range fixes {
		fixes[i] = PositionFix{Latitude: float64(i), Longitude: 0}
	}
	out := compactPositionTrail(fixes, PositionTrailCompactTo)
	if len(out) > PositionTrailCompactTo {
		t.Fatalf("fixes = %d, want <= %d", len(out), PositionTrailCompactTo)
	}
}
