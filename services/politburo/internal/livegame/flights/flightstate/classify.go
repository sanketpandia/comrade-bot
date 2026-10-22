package flightstate

import (
	"time"
)

// Classify updates estimator state from history window features.
func Classify(prev State, h History, cfg Config, now time.Time) State {
	w := WindowFeaturesFrom(h, cfg, now)
	cur := w.Current
	next := prev
	next.Reasons = nil
	next.Features = snapshotFeatures(w)

	if len(h.Samples) == 0 || !cur.coreValid() {
		next.Phase = PhaseUnknown
		next.Activity = ActivityUnknown
		next.DataQuality = DataQualityInvalid
		next.Confidence = 0.1
		next.Reasons = []string{"invalid_measurements"}
		next.APPlus = APPlus{}
		return next
	}

	next.GroundRefAltFt = updateGroundRef(prev, cur, cfg)

	reportTime := cur.LastReport
	if !cur.LastReportValid {
		reportTime = cur.ObservedAt
	}
	staleAge := now.Sub(reportTime)
	if staleAge > cfg.StaleHoldMax {
		next.Phase = PhaseUnknown
		next.DataQuality = DataQualityStale
		next.Confidence = 0.15
		next.Reasons = []string{"report_stale"}
		next.APPlus = APPlus{}
		return next
	}
	if staleAge > cfg.StaleAfter {
		next.DataQuality = DataQualityStale
		next.Reasons = append(next.Reasons, "stale_hold")
	} else if w.Pair.GapClass == GapLong {
		next.DataQuality = DataQualityLongGap
	} else if !cur.IsConnected {
		next.DataQuality = DataQualityUncertain
	} else {
		next.DataQuality = DataQualityFresh
	}

	if w.Pair.GapClass == GapTooLarge {
		next.Reasons = append(next.Reasons, "report_gap_too_large")
	}

	if prev.Phase == PhaseSpawnedIn && len(h.Samples) == 1 {
		next.Activity = activityFromPair(w.Pair)
		return commitPhase(next, PhaseSpawnedIn, 0.35, []string{"first_sample"}, now)
	}

	if prev.Phase == PhaseSpawnedIn && staleAge > cfg.SpawnedInMax {
		return commitPhase(next, PhaseUnknown, 0.2, []string{"insufficient_observations"}, now)
	}

	ap := mergeAPPlusPilotState(scoreAPPlus(prev, w, cfg, now), prev, w, now)
	next.APPlus = ap
	if ap.Candidate && cfg.APPlusEmitPhase {
		return commitPhase(next, PhaseAwaitingAPPlus, ap.Score, ap.Reasons, now)
	}

	candidates := []phaseCandidate{
		scoreParked(prev, w, cfg),
		scoreTaxiing(prev, w, cfg),
		scoreTakeoff(prev, w, cfg),
		scoreClimbing(prev, w, cfg),
		scoreCruise(prev, w, cfg),
		scoreDescending(prev, w, cfg),
		scoreLanding(prev, w, cfg),
	}

	if w.Pair.GapClass == GapTooLarge && (prev.Phase == PhaseTakeoff || prev.Phase == PhaseLanding) {
		next.Activity = activityFromPair(w.Pair)
		next.Reasons = append(next.Reasons, "gap_blocks_landing_takeoff_inference")
		return applyHysteresis(prev, next, prev.Phase, 0.4, next.Reasons, cfg, now)
	}

	best := phaseCandidate{phase: prev.Phase, strength: 0.35}
	for _, c := range candidates {
		if c.strength > best.strength {
			best = c
		}
	}

	if best.strength < 0.45 && next.DataQuality == DataQualityFresh {
		next.Activity = activityFromPair(w.Pair)
		next.Confidence = prev.Confidence * 0.95
		if next.Confidence < 0.2 {
			next.Confidence = 0.2
		}
		next.Reasons = append(next.Reasons, "weak_evidence_hold")
		return next
	}

	if staleAge > cfg.StaleAfter && staleAge <= cfg.StaleHoldMax {
		next.Activity = activityFromPair(w.Pair)
		next.Confidence = prev.Confidence * 0.9
		return next
	}

	next.Activity = activityFromPair(w.Pair)
	return applyHysteresis(prev, next, best.phase, best.strength, best.reasons, cfg, now)
}

func activityFromPair(p PairFeatures) Activity {
	if p.ElapsedSec == 0 {
		return ActivityUnknown
	}
	if p.Moved {
		return ActivityMoving
	}
	return ActivityStationary
}

func snapshotFeatures(w WindowFeatures) WindowFeatureSnapshot {
	return WindowFeatureSnapshot{
		ElapsedSec:            w.Pair.ElapsedSec,
		DistanceNM:            w.Pair.DistanceNM,
		AltDeltaFt:            w.Pair.AltDeltaFt,
		AvgVerticalFpm:        w.Pair.AvgVerticalFpm,
		SpeedDeltaKt:          w.Pair.SpeedDeltaKt,
		TrackDeltaDeg:         w.Pair.TrackDeltaDeg,
		Moved:                 w.Pair.Moved,
		AltStable:             w.Pair.AltStable,
		ProgressRatio:         w.Pair.ProgressRatio,
		GapClass:              w.Pair.GapClass,
		Fresh:                 w.Pair.Fresh,
		StationarySamples:     w.StationarySamples,
		StationaryDurationSec: w.StationaryDurationSec,
		SustainedClimbCount:   w.SustainedClimbCount,
		SustainedDescentCount: w.SustainedDescentCount,
		LastReportAdvancing:   w.LastReportAdvancing,
	}
}

func applyHysteresis(prev, next State, target Phase, strength float64, reasons []string, cfg Config, now time.Time) State {
	if target == prev.Phase {
		next.Confidence = strength
		if len(reasons) > 0 {
			next.Reasons = reasons
		}
		next.Pending = nil
		return next
	}
	if strength >= cfg.StrongEvidence {
		return commitPhase(next, target, strength, reasons, now)
	}
	need := confirmationsFor(cfg, prev.Phase, target)
	if prev.Pending != nil && prev.Pending.Phase == target {
		count := prev.Pending.Count + 1
		if count >= need {
			return commitPhase(next, target, strength, reasons, now)
		}
		p := *prev.Pending
		p.Count = count
		next.Pending = &p
		next.Confidence = prev.Confidence
		return next
	}
	next.Pending = &Pending{Phase: target, Count: 1, Since: now}
	next.Confidence = prev.Confidence
	return next
}

func commitPhase(st State, phase Phase, confidence float64, reasons []string, now time.Time) State {
	if st.Phase != phase {
		st.PreviousPhase = st.Phase
		st.Phase = phase
		st.PhaseSince = now
		st.Pending = nil
	}
	st.Confidence = confidence
	if len(reasons) > 0 {
		st.Reasons = reasons
	}
	return st
}

const apPlusPilotStateScore = 0.95

func mergeAPPlusPilotState(ap APPlus, prev State, w WindowFeatures, now time.Time) APPlus {
	if w.Current.PilotState != PilotStateInBackground {
		return ap
	}
	ap.Candidate = true
	if ap.Score < apPlusPilotStateScore {
		ap.Score = apPlusPilotStateScore
	}
	const reason = "pilot_state_in_background"
	if !sliceContains(ap.Reasons, reason) {
		ap.Reasons = append(append([]string(nil), ap.Reasons...), reason)
	}
	if ap.Since.IsZero() {
		if prev.APPlus.Candidate && !prev.APPlus.Since.IsZero() {
			ap.Since = prev.APPlus.Since
		} else {
			ap.Since = now
		}
	}
	return ap
}

func sliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func scoreAPPlus(prev State, w WindowFeatures, cfg Config, now time.Time) APPlus {
	out := APPlus{}
	if !w.Current.IsConnected || !w.LastReportAdvancing {
		return out
	}
	if stale := now.Sub(w.Current.LastReport); stale > cfg.StaleAfter {
		return out
	}
	airbornePhase := prev.Phase == PhaseCruise || prev.Phase == PhaseClimbing || prev.Phase == PhaseDescending
	if !airbornePhase && !w.AirborneInWindow {
		return out
	}
	alt := w.Current.Altitude
	above := altAboveGround(alt, prev.GroundRefAltFt)
	highEnough := alt >= cfg.APPlusMinAltFt || above >= 5000
	if !highEnough {
		return out
	}
	if w.StationarySamples < cfg.APPlusMinStationarySamples {
		return out
	}
	if w.StationaryDurationSec < cfg.APPlusMinStasis.Seconds() {
		return out
	}
	if w.AltDeltaWindowFt > cfg.APPlusMaxAltDeltaFt {
		return out
	}

	score := cfg.APPlusScoreWeights.Stationary
	reasons := []string{"stationary_high_alt"}
	if w.Pair.AltStable {
		score += cfg.APPlusScoreWeights.AltStable
		reasons = append(reasons, "alt_stable")
	}
	if w.Current.SpeedValid && w.Current.Speed < 30 {
		score += cfg.APPlusScoreWeights.SpeedLow
	}
	if w.Pair.ProgressRatio < 0.1 && w.Current.SpeedValid && w.Current.Speed > 50 {
		score += cfg.APPlusScoreWeights.ProgressMismatch
		reasons = append(reasons, "progress_mismatch")
	}
	if score < 0.5 {
		return out
	}
	out.Candidate = true
	out.Score = score
	if prev.APPlus.Since.IsZero() {
		out.Since = now
	} else {
		out.Since = prev.APPlus.Since
	}
	out.Reasons = reasons
	return out
}
