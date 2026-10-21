package flightstate

import "math"

func nearGround(alt float64, groundRef float64, cfg Config) bool {
	if groundRef > 0 {
		return alt <= groundRef+cfg.NearGroundMarginFt
	}
	return alt <= cfg.FallbackGroundMaxFt
}

func altAboveGround(alt float64, groundRef float64) float64 {
	if groundRef > 0 {
		return alt - groundRef
	}
	return alt
}

func updateGroundRef(st State, s Sample, cfg Config) float64 {
	if !s.coreValid() {
		return st.GroundRefAltFt
	}
	if groundLikeSample(s, cfg) && s.Speed <= cfg.ParkedMaxSpeedKt+5 {
		if st.GroundRefAltFt == 0 {
			return s.Altitude
		}
		// Slowly adapt ground reference toward stable ground samples.
		return (st.GroundRefAltFt * 0.7) + (s.Altitude * 0.3)
	}
	return st.GroundRefAltFt
}

func decelerating(pair PairFeatures) bool {
	return pair.SpeedDeltaKt < -5
}

func absF(v float64) float64 {
	return math.Abs(v)
}
