package flightstate

import "time"

// History holds recent samples and the latest full estimator state for one flight.
type History struct {
	Samples []Sample `json:"samples"`
	State   State    `json:"state"`
}

// Push appends a sample when it advances lastReport; returns a copy and an optional reject reason.
func (h History) Push(sample Sample, cfg Config) (History, string) {
	out := h.clone()
	if !sample.LastReportValid {
		return out, "invalid_last_report"
	}
	if len(out.Samples) > 0 {
		last := out.Samples[len(out.Samples)-1]
		if sample.LastReport.Equal(last.LastReport) {
			return out, "duplicate_last_report"
		}
		if !sample.LastReport.After(last.LastReport) {
			return out, "non_positive_elapsed"
		}
	}
	out.Samples = append(out.Samples, sample)
	out.trim(cfg)
	if out.State.Phase == "" {
		out.State = initialState(sample.LastReport)
	}
	return out, ""
}

func initialState(at time.Time) State {
	return State{
		Phase:       PhaseSpawnedIn,
		Activity:    ActivityUnknown,
		DataQuality: DataQualityFresh,
		Confidence:  0.3,
		PhaseSince:  at,
	}
}

func (h *History) trim(cfg Config) {
	cutoff := time.Time{}
	if cfg.HistoryWindow > 0 && len(h.Samples) > 0 {
		latest := h.Samples[len(h.Samples)-1].LastReport
		cutoff = latest.Add(-cfg.HistoryWindow)
	}
	filtered := make([]Sample, 0, len(h.Samples))
	for _, s := range h.Samples {
		if cutoff.IsZero() || !s.LastReport.Before(cutoff) {
			filtered = append(filtered, s)
		}
	}
	h.Samples = filtered
	if cfg.HistorySize > 0 && len(h.Samples) > cfg.HistorySize {
		h.Samples = h.Samples[len(h.Samples)-cfg.HistorySize:]
	}
}

func (h History) clone() History {
	out := History{State: h.State}
	if h.State.Pending != nil {
		p := *h.State.Pending
		out.State.Pending = &p
	}
	if len(h.State.Reasons) > 0 {
		out.State.Reasons = append([]string(nil), h.State.Reasons...)
	}
	if len(h.State.APPlus.Reasons) > 0 {
		out.State.APPlus.Reasons = append([]string(nil), h.State.APPlus.Reasons...)
	}
	if len(h.Samples) > 0 {
		out.Samples = append([]Sample(nil), h.Samples...)
	}
	return out
}

// WithState returns a copy whose embedded state matches the given full state.
func (h History) WithState(st State) History {
	out := h.clone()
	out.State = st
	if st.Pending != nil {
		p := *st.Pending
		out.State.Pending = &p
	}
	if len(st.Reasons) > 0 {
		out.State.Reasons = append([]string(nil), st.Reasons...)
	}
	if len(st.APPlus.Reasons) > 0 {
		out.State.APPlus.Reasons = append([]string(nil), st.APPlus.Reasons...)
	}
	return out
}
