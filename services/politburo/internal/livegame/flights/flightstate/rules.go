package flightstate

type phaseCandidate struct {
	phase    Phase
	strength float64
	reasons  []string
}

func scoreParked(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !s.coreValid() || !nearGround(s.Altitude, prev.GroundRefAltFt, cfg) {
		return phaseCandidate{phase: PhaseParked}
	}
	if s.Speed > cfg.ParkedMaxSpeedKt || w.Pair.Moved {
		return phaseCandidate{phase: PhaseParked}
	}
	if w.StationarySamples < 2 {
		return phaseCandidate{phase: PhaseParked}
	}
	return phaseCandidate{phase: PhaseParked, strength: 0.8, reasons: []string{"ground_stationary"}}
}

func scoreTaxiing(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !s.coreValid() || !nearGround(s.Altitude, prev.GroundRefAltFt, cfg) {
		return phaseCandidate{phase: PhaseTaxiing}
	}
	if s.Speed < cfg.TaxiMinSpeedKt || s.Speed > cfg.TaxiMaxSpeedKt {
		return phaseCandidate{phase: PhaseTaxiing}
	}
	if !w.Pair.Moved && w.Pair.DistanceNM <= cfg.MovementTolNM {
		return phaseCandidate{phase: PhaseTaxiing}
	}
	return phaseCandidate{phase: PhaseTaxiing, strength: 0.75, reasons: []string{"ground_taxi"}}
}

func scoreTakeoff(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !w.HasPair || !s.coreValid() {
		return phaseCandidate{phase: PhaseTakeoff}
	}
	groundLike := prev.Phase == PhaseParked || prev.Phase == PhaseTaxiing || w.GroundLikeInWindow
	if !groundLike {
		return phaseCandidate{phase: PhaseTakeoff}
	}
	above := altAboveGround(s.Altitude, prev.GroundRefAltFt)
	vsOK := s.VerticalSpeedValid && s.VerticalSpeed >= cfg.TakeoffMinVSFpm
	if s.Speed >= cfg.TakeoffMinSpeedKt && (above >= cfg.TakeoffMinAltAboveGnd || vsOK) {
		return phaseCandidate{phase: PhaseTakeoff, strength: 0.82, reasons: []string{"takeoff_roll"}}
	}
	return phaseCandidate{phase: PhaseTakeoff}
}

func scoreClimbing(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !s.coreValid() {
		return phaseCandidate{phase: PhaseClimbing}
	}
	above := altAboveGround(s.Altitude, prev.GroundRefAltFt)
	if above < cfg.TakeoffExitAltAboveGnd && prev.Phase == PhaseTakeoff {
		return phaseCandidate{phase: PhaseClimbing}
	}
	vs := s.VerticalSpeed
	if s.VerticalSpeedValid && vs >= cfg.ClimbEnterVSFpm && w.SustainedClimbCount >= 2 {
		return phaseCandidate{phase: PhaseClimbing, strength: 0.78, reasons: []string{"sustained_climb"}}
	}
	if prev.Phase == PhaseClimbing && w.Pair.AvgVerticalFpm >= cfg.ClimbExitAvgFpm {
		return phaseCandidate{phase: PhaseClimbing, strength: 0.65, reasons: []string{"climb_hold"}}
	}
	return phaseCandidate{phase: PhaseClimbing}
}

func scoreDescending(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !s.coreValid() {
		return phaseCandidate{phase: PhaseDescending}
	}
	vs := s.VerticalSpeed
	if s.VerticalSpeedValid && vs <= cfg.DescendEnterVSFpm && w.SustainedDescentCount >= 2 {
		return phaseCandidate{phase: PhaseDescending, strength: 0.78, reasons: []string{"sustained_descent"}}
	}
	if prev.Phase == PhaseDescending && w.Pair.AvgVerticalFpm <= cfg.DescendExitAvgFpm {
		return phaseCandidate{phase: PhaseDescending, strength: 0.65, reasons: []string{"descent_hold"}}
	}
	return phaseCandidate{phase: PhaseDescending}
}

func scoreCruise(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !w.HasPair || !s.coreValid() {
		return phaseCandidate{phase: PhaseCruise}
	}
	above := altAboveGround(s.Altitude, prev.GroundRefAltFt)
	airborne := above >= cfg.CruiseMinAltAboveGnd || (prev.GroundRefAltFt == 0 && s.Altitude >= cfg.CruiseFallbackAltFt)
	if !airborne {
		return phaseCandidate{phase: PhaseCruise}
	}
	if absF(w.Pair.AvgVerticalFpm) > cfg.CruiseMaxAbsAvgFpm {
		return phaseCandidate{phase: PhaseCruise}
	}
	if !w.Pair.Moved || w.Pair.ProgressRatio < cfg.CruiseMinProgress {
		return phaseCandidate{phase: PhaseCruise}
	}
	if w.SustainedClimbCount > 0 || w.SustainedDescentCount > 0 {
		return phaseCandidate{phase: PhaseCruise}
	}
	return phaseCandidate{phase: PhaseCruise, strength: 0.72, reasons: []string{"level_progress"}}
}

func scoreLanding(prev State, w WindowFeatures, cfg Config) phaseCandidate {
	s := w.Current
	if !w.HasPair || !s.coreValid() {
		return phaseCandidate{phase: PhaseLanding}
	}
	if w.Pair.GapClass == GapTooLarge {
		return phaseCandidate{phase: PhaseLanding}
	}
	fromDescent := prev.Phase == PhaseDescending || w.AirborneInWindow
	if !fromDescent {
		return phaseCandidate{phase: PhaseLanding}
	}
	if nearGround(s.Altitude, prev.GroundRefAltFt, cfg) && decelerating(w.Pair) {
		return phaseCandidate{phase: PhaseLanding, strength: 0.8, reasons: []string{"approach_decel"}}
	}
	return phaseCandidate{phase: PhaseLanding}
}

func expectedNext(prev Phase) []Phase {
	switch prev {
	case PhaseSpawnedIn:
		return []Phase{PhaseTaxiing, PhaseParked, PhaseTakeoff, PhaseClimbing, PhaseUnknown}
	case PhaseParked:
		return []Phase{PhaseTaxiing, PhaseTakeoff, PhaseUnknown}
	case PhaseTaxiing:
		return []Phase{PhaseTakeoff, PhaseParked, PhaseUnknown}
	case PhaseTakeoff:
		return []Phase{PhaseClimbing, PhaseUnknown}
	case PhaseClimbing:
		return []Phase{PhaseCruise, PhaseDescending, PhaseUnknown}
	case PhaseCruise:
		return []Phase{PhaseDescending, PhaseClimbing, PhaseUnknown}
	case PhaseDescending:
		return []Phase{PhaseLanding, PhaseCruise, PhaseUnknown}
	case PhaseLanding:
		return []Phase{PhaseTaxiing, PhaseParked, PhaseUnknown}
	case PhaseAwaitingAPPlus:
		return []Phase{PhaseClimbing, PhaseCruise, PhaseDescending, PhaseTaxiing, PhaseUnknown}
	default:
		return nil
	}
}

func isExpectedTransition(from, to Phase, cfg Config) bool {
	if from == to {
		return true
	}
	for _, p := range expectedNext(from) {
		if p == to {
			return true
		}
	}
	return false
}

func confirmationsFor(cfg Config, prev Phase, next Phase) int {
	base := 2
	if cfg.Confirmations != nil {
		if n, ok := cfg.Confirmations[next]; ok {
			base = n
		}
	}
	if !isExpectedTransition(prev, next, cfg) {
		base += cfg.UnexpectedTransitionConfirmations
	}
	return base
}
