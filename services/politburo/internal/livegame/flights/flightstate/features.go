package flightstate

import (
	"math"
	"time"
)

// PairFeatures compares two consecutive samples.
type PairFeatures struct {
	ElapsedSec     float64
	DistanceNM     float64
	AltDeltaFt     float64
	AvgVerticalFpm float64
	SpeedDeltaKt   float64
	TrackDeltaDeg  float64
	Moved          bool
	AltStable      bool
	ProgressRatio  float64
	GapClass       GapClass
	Fresh          bool
}

// WindowFeatures aggregates the history window for classification.
type WindowFeatures struct {
	Pair                  PairFeatures
	HasPair               bool
	Current               Sample
	Previous              Sample
	StationarySamples     int
	StationaryDurationSec float64
	SustainedClimbCount   int
	SustainedDescentCount int
	LastReportAdvancing   bool
	GroundLikeInWindow    bool
	AirborneInWindow      bool
	AltDeltaWindowFt      float64
}

// WindowFeaturesFrom builds window features from history and config.
func WindowFeaturesFrom(h History, cfg Config, now time.Time) WindowFeatures {
	w := WindowFeatures{Current: latestSample(h)}
	if len(h.Samples) >= 2 {
		prev := h.Samples[len(h.Samples)-2]
		cur := h.Samples[len(h.Samples)-1]
		w.Previous = prev
		w.Current = cur
		w.HasPair = true
		w.Pair = PairFeaturesBetween(prev, cur, cfg, now)
		w.LastReportAdvancing = cur.LastReport.After(prev.LastReport)
	}
	w.aggregateCounts(h, cfg)
	return w
}

func latestSample(h History) Sample {
	if len(h.Samples) == 0 {
		return Sample{}
	}
	return h.Samples[len(h.Samples)-1]
}

// PairFeaturesBetween computes features between prev and cur.
func PairFeaturesBetween(prev, cur Sample, cfg Config, now time.Time) PairFeatures {
	p := PairFeatures{Fresh: true}
	if !prev.LastReportValid || !cur.LastReportValid {
		return p
	}
	elapsed := cur.LastReport.Sub(prev.LastReport).Seconds()
	p.ElapsedSec = elapsed
	if elapsed <= 0 {
		return p
	}
	if elapsed >= cfg.TooLargeGapAfter.Seconds() {
		p.GapClass = GapTooLarge
	} else if elapsed >= cfg.LongGapAfter.Seconds() {
		p.GapClass = GapLong
	} else {
		p.GapClass = GapNormal
	}
	if cur.coreValid() && prev.coreValid() {
		p.DistanceNM = haversineNM(prev.Latitude, prev.Longitude, cur.Latitude, cur.Longitude)
		p.Moved = p.DistanceNM > cfg.MovementTolNM
		p.AltDeltaFt = cur.Altitude - prev.Altitude
		p.AltStable = math.Abs(p.AltDeltaFt) < 50
		minutes := elapsed / 60
		if minutes > 0 {
			p.AvgVerticalFpm = p.AltDeltaFt / minutes
		}
		if prev.SpeedValid && cur.SpeedValid {
			p.SpeedDeltaKt = cur.Speed - prev.Speed
		}
		if prev.TrackValid && cur.TrackValid {
			p.TrackDeltaDeg = shortestAngleDelta(prev.Track, cur.Track)
		}
		if cur.SpeedValid && elapsed > 0 {
			impliedNM := (cur.Speed / 3600) * elapsed
			if impliedNM > 0.001 {
				p.ProgressRatio = p.DistanceNM / impliedNM
			}
		}
	}
	if cur.LastReportValid && now.Sub(cur.LastReport) > cfg.StaleAfter {
		p.Fresh = false
	}
	return p
}

func (w *WindowFeatures) aggregateCounts(h History, cfg Config) {
	if len(h.Samples) == 0 {
		return
	}
	minAlt, maxAlt := h.Samples[0].Altitude, h.Samples[0].Altitude
	for i := len(h.Samples) - 1; i >= 0; i-- {
		s := h.Samples[i]
		if s.AltitudeValid {
			if s.Altitude < minAlt {
				minAlt = s.Altitude
			}
			if s.Altitude > maxAlt {
				maxAlt = s.Altitude
			}
		}
		if i == 0 {
			continue
		}
		prev := h.Samples[i-1]
		pair := PairFeaturesBetween(prev, s, cfg, s.LastReport)
		if !pair.Moved {
			w.StationarySamples++
			w.StationaryDurationSec += pair.ElapsedSec
		}
		if pair.AvgVerticalFpm >= cfg.ClimbEnterAvgFpm {
			w.SustainedClimbCount++
		} else {
			w.SustainedClimbCount = 0
		}
		if pair.AvgVerticalFpm <= cfg.DescendEnterAvgFpm {
			w.SustainedDescentCount++
		} else {
			w.SustainedDescentCount = 0
		}
		if groundLikeSample(s, cfg) {
			w.GroundLikeInWindow = true
		}
		if airborneSample(s, h.State.GroundRefAltFt, cfg) {
			w.AirborneInWindow = true
		}
	}
	w.AltDeltaWindowFt = maxAlt - minAlt
}

func haversineNM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusNM = 3440.065
	rad := math.Pi / 180
	φ1, φ2 := lat1*rad, lat2*rad
	dφ := (lat2 - lat1) * rad
	dλ := (lon2 - lon1) * rad
	a := math.Sin(dφ/2)*math.Sin(dφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusNM * c
}

func shortestAngleDelta(a, b float64) float64 {
	d := math.Mod(b-a+540, 360) - 180
	return d
}

func groundLikeSample(s Sample, cfg Config) bool {
	if !s.coreValid() {
		return false
	}
	lowSpeed := s.Speed <= cfg.TaxiMaxSpeedKt
	lowVS := !s.VerticalSpeedValid || math.Abs(s.VerticalSpeed) < 200
	return lowSpeed && lowVS
}

func airborneSample(s Sample, groundRef float64, cfg Config) bool {
	if !s.AltitudeValid {
		return false
	}
	if groundRef > 0 {
		return s.Altitude > groundRef+cfg.CruiseMinAltAboveGnd/4
	}
	return s.Altitude > cfg.CruiseFallbackAltFt
}
