package flights

import (
	"math"
	"time"

	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

const (
	PositionTrailMax          = 200
	PositionTrailCompactTo    = 150
	PositionTrailKeepRecent   = 20
	PositionFixInterval       = 5 * time.Minute
)

// PositionFix is one point on the long position trail.
type PositionFix struct {
	At        time.Time `json:"at"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude"`
	Speed     float64   `json:"speed"`
}

// PositionTrail is the cached flown path for one active flight.
type PositionTrail struct {
	Fixes []PositionFix `json:"fixes,omitempty"`
}

// PositionFixDue reports whether a new fix should be appended at now.
func PositionFixDue(trail PositionTrail, now time.Time) bool {
	if len(trail.Fixes) == 0 {
		return true
	}
	last := trail.Fixes[len(trail.Fixes)-1]
	return !last.At.IsZero() && now.Sub(last.At) >= PositionFixInterval
}

// PositionTrailPhaseActive reports whether the estimator phase warrants trail points.
func PositionTrailPhaseActive(phase flightstate.Phase) bool {
	switch phase {
	case flightstate.PhaseTaxiing, flightstate.PhaseTakeoff, flightstate.PhaseClimbing,
		flightstate.PhaseCruise, flightstate.PhaseDescending, flightstate.PhaseLanding:
		return true
	default:
		return false
	}
}

// AppendPositionFix adds a fix when phase is active moving and the 5-minute gap is met.
func AppendPositionFix(trail PositionTrail, phase flightstate.Phase, activity flightstate.Activity, obs Observation, now time.Time) (PositionTrail, bool) {
	if !PositionTrailPhaseActive(phase) || activity != flightstate.ActivityMoving {
		return trail, false
	}
	if !PositionFixDue(trail, now) {
		return trail, false
	}
	fix := PositionFix{
		At:        now.UTC(),
		Latitude:  obs.Latitude,
		Longitude: obs.Longitude,
		Altitude:  obs.Altitude,
		Speed:     obs.Speed,
	}
	out := trail
	out.Fixes = append(append([]PositionFix(nil), out.Fixes...), fix)
	if len(out.Fixes) > PositionTrailMax {
		out.Fixes = compactPositionTrail(out.Fixes, PositionTrailCompactTo)
	}
	return out, true
}

func compactPositionTrail(fixes []PositionFix, target int) []PositionFix {
	if len(fixes) <= target {
		return fixes
	}
	keep := make(map[int]bool, target)
	keep[0] = true
	keep[len(fixes)-1] = true
	recentStart := len(fixes) - PositionTrailKeepRecent
	if recentStart < 0 {
		recentStart = 0
	}
	for i := recentStart; i < len(fixes); i++ {
		keep[i] = true
	}
	for len(keep) > target {
		bestIdx := -1
		bestDist := math.MaxFloat64
		for i := 1; i < len(fixes)-1; i++ {
			if keep[i] {
				continue
			}
			d := pointToSegmentDistanceNM(
				fixes[i].Latitude, fixes[i].Longitude,
				fixes[i-1].Latitude, fixes[i-1].Longitude,
				fixes[i+1].Latitude, fixes[i+1].Longitude,
			)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		delete(keep, bestIdx)
	}
	out := make([]PositionFix, 0, len(keep))
	for i, fix := range fixes {
		if keep[i] {
			out = append(out, fix)
		}
	}
	return out
}

func pointToSegmentDistanceNM(lat, lon, latA, lonA, latB, lonB float64) float64 {
	dx := lonB - lonA
	dy := latB - latA
	if dx == 0 && dy == 0 {
		return haversineNM(lat, lon, latA, lonA)
	}
	t := ((lon-lonA)*dx + (lat-latA)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		return haversineNM(lat, lon, latA, lonA)
	}
	if t > 1 {
		return haversineNM(lat, lon, latB, lonB)
	}
	projLat := latA + t*(latB-latA)
	projLon := lonA + t*(lonB-lonA)
	return haversineNM(lat, lon, projLat, projLon)
}

func haversineNM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusNM = 3440.065
	rad := math.Pi / 180
	φ1, φ2 := lat1*rad, lat2*rad
	dφ := (lat2 - lat1) * rad
	dλ := (lon2 - lon1) * rad
	a := math.Sin(dφ/2)*math.Sin(dφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	return earthRadiusNM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
