package flightstate

import "time"

// Config holds tunable thresholds for temporal flight-phase estimation.
// Field comments describe tuning hypotheses, not Infinite Flight API limits.
type Config struct {
	HistorySize      int
	HistoryWindow    time.Duration
	MovementTolNM    float64
	StaleAfter       time.Duration
	StaleHoldMax     time.Duration
	SpawnedInMax     time.Duration
	LongGapAfter     time.Duration
	TooLargeGapAfter time.Duration

	NearGroundMarginFt     float64
	FallbackGroundMaxFt    float64
	ParkedMaxSpeedKt       float64
	TaxiMinSpeedKt         float64
	TaxiMaxSpeedKt         float64
	TakeoffMinSpeedKt      float64
	TakeoffMinAltAboveGnd  float64
	TakeoffMinVSFpm        float64
	TakeoffExitAltAboveGnd float64
	ClimbEnterVSFpm        float64
	ClimbEnterAvgFpm       float64
	ClimbExitAvgFpm        float64
	DescendEnterVSFpm      float64
	DescendEnterAvgFpm     float64
	DescendExitAvgFpm      float64
	CruiseMinAltAboveGnd   float64
	CruiseFallbackAltFt    float64
	CruiseMaxAbsAvgFpm     float64
	CruiseMinProgress      float64
	LandingMaxSpeedKt      float64

	StrongEvidence                    float64
	UnexpectedTransitionConfirmations int
	Confirmations                     map[Phase]int

	APPlusEmitPhase            bool
	APPlusMinAltFt             float64
	APPlusMinStasis            time.Duration
	APPlusMinStationarySamples int
	APPlusMaxAltDeltaFt        float64
	APPlusScoreWeights         APPlusScoreWeights
}

// APPlusScoreWeights tune the AP+ candidate score (not calibrated probabilities).
type APPlusScoreWeights struct {
	Stationary       float64
	AltStable        float64
	SpeedLow         float64
	ProgressMismatch float64
}

// DefaultConfig returns production-default estimator thresholds.
func DefaultConfig() Config {
	return Config{
		HistorySize:      10,
		HistoryWindow:    15 * time.Minute,
		MovementTolNM:    0.05,
		StaleAfter:       3 * time.Minute,
		StaleHoldMax:     8 * time.Minute,
		SpawnedInMax:     3 * time.Minute,
		LongGapAfter:     2 * time.Minute,
		TooLargeGapAfter: 5 * time.Minute,

		NearGroundMarginFt:     200,
		FallbackGroundMaxFt:    500,
		ParkedMaxSpeedKt:       3,
		TaxiMinSpeedKt:         3,
		TaxiMaxSpeedKt:         40,
		TakeoffMinSpeedKt:      80,
		TakeoffMinAltAboveGnd:  100,
		TakeoffMinVSFpm:        300,
		TakeoffExitAltAboveGnd: 1500,
		ClimbEnterVSFpm:        500,
		ClimbEnterAvgFpm:       300,
		ClimbExitAvgFpm:        200,
		DescendEnterVSFpm:      -500,
		DescendEnterAvgFpm:     -300,
		DescendExitAvgFpm:      -200,
		CruiseMinAltAboveGnd:   2000,
		CruiseFallbackAltFt:    5000,
		CruiseMaxAbsAvgFpm:     300,
		CruiseMinProgress:      0.5,
		LandingMaxSpeedKt:      40,

		StrongEvidence:                    0.85,
		UnexpectedTransitionConfirmations: 3,
		Confirmations: map[Phase]int{
			PhaseSpawnedIn:      1,
			PhaseTaxiing:        2,
			PhaseTakeoff:        2,
			PhaseClimbing:       2,
			PhaseCruise:         2,
			PhaseDescending:     2,
			PhaseLanding:        2,
			PhaseParked:         2,
			PhaseAwaitingAPPlus: 2,
			PhaseUnknown:        1,
		},

		APPlusEmitPhase:            false,
		APPlusMinAltFt:             10000,
		APPlusMinStasis:            3 * time.Minute,
		APPlusMinStationarySamples: 3,
		APPlusMaxAltDeltaFt:        100,
		APPlusScoreWeights: APPlusScoreWeights{
			Stationary:       0.35,
			AltStable:        0.25,
			SpeedLow:         0.2,
			ProgressMismatch: 0.2,
		},
	}
}
