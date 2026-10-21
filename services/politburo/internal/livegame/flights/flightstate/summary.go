package flightstate

import (
	"math"
	"time"
)

// Summary is the flight-phase output stored on the enriched record (no Pending/Features).
type Summary struct {
	Phase          Phase                  `json:"phase"`
	Activity       Activity               `json:"activity"`
	DataQuality    DataQuality            `json:"dataQuality"`
	Confidence     float64                `json:"confidence"`
	PreviousPhase  Phase                  `json:"previousPhase,omitempty"`
	PhaseSince     time.Time              `json:"phaseSince"`
	GroundRefAltFt float64                `json:"groundRefAltFt,omitempty"`
	Reasons        []string               `json:"reasons,omitempty"`
	APPlus         APPlus                 `json:"apPlus"`
	Features       *WindowFeatureSnapshot `json:"features,omitempty"`
}

// SummaryFromState builds a cache-safe summary from full state.
func SummaryFromState(st State, includeFeatures bool) Summary {
	sum := Summary{
		Phase:          st.Phase,
		Activity:       st.Activity,
		DataQuality:    st.DataQuality,
		Confidence:     quantizeConfidence(st.Confidence),
		PreviousPhase:  st.PreviousPhase,
		PhaseSince:     st.PhaseSince,
		GroundRefAltFt: st.GroundRefAltFt,
		Reasons:        append([]string(nil), st.Reasons...),
		APPlus: APPlus{
			Candidate: st.APPlus.Candidate,
			Score:     quantizeConfidence(st.APPlus.Score),
			Since:     st.APPlus.Since,
			Reasons:   append([]string(nil), st.APPlus.Reasons...),
		},
	}
	if includeFeatures && (st.Features.ElapsedSec != 0 || st.Features.DistanceNM != 0 || st.Features.GapClass != "") {
		f := st.Features
		sum.Features = &f
	}
	return sum
}

// Clone deep-copies summary slices.
func (s Summary) Clone() Summary {
	out := s
	if len(s.Reasons) > 0 {
		out.Reasons = append([]string(nil), s.Reasons...)
	}
	if len(s.APPlus.Reasons) > 0 {
		out.APPlus.Reasons = append([]string(nil), s.APPlus.Reasons...)
	}
	if s.Features != nil {
		f := *s.Features
		out.Features = &f
	}
	return out
}

// Equal reports whether two summaries match for cache dirty checks.
func (s Summary) Equal(other Summary) bool {
	if s.Phase != other.Phase || s.Activity != other.Activity || s.DataQuality != other.DataQuality ||
		s.Confidence != other.Confidence || s.PreviousPhase != other.PreviousPhase ||
		s.GroundRefAltFt != other.GroundRefAltFt || s.APPlus.Candidate != other.APPlus.Candidate ||
		s.APPlus.Score != other.APPlus.Score {
		return false
	}
	if !s.PhaseSince.Equal(other.PhaseSince) || !s.APPlus.Since.Equal(other.APPlus.Since) {
		return false
	}
	if !stringSliceEqual(s.Reasons, other.Reasons) || !stringSliceEqual(s.APPlus.Reasons, other.APPlus.Reasons) {
		return false
	}
	if (s.Features == nil) != (other.Features == nil) {
		return false
	}
	if s.Features != nil && *s.Features != *other.Features {
		return false
	}
	return true
}

func quantizeConfidence(v float64) float64 {
	const step = 0.05
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v/step) * step
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
