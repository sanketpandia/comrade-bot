package flightstate

import "time"

// Phase is the estimated flight segment.
type Phase string

const (
	PhaseSpawnedIn      Phase = "spawned_in"
	PhaseTaxiing        Phase = "taxiing"
	PhaseTakeoff        Phase = "takeoff"
	PhaseClimbing       Phase = "climbing"
	PhaseCruise         Phase = "cruise"
	PhaseDescending     Phase = "descending"
	PhaseLanding        Phase = "landing"
	PhaseParked         Phase = "parked"
	PhaseAwaitingAPPlus Phase = "awaiting_ap_plus_resumption"
	PhaseUnknown        Phase = "unknown"
)

// Activity describes horizontal motion.
type Activity string

const (
	ActivityMoving     Activity = "moving"
	ActivityStationary Activity = "stationary"
	ActivityUnknown    Activity = "unknown"
)

// DataQuality captures measurement freshness and trust.
type DataQuality string

const (
	DataQualityFresh     DataQuality = "fresh"
	DataQualityLongGap   DataQuality = "long_gap"
	DataQualityStale     DataQuality = "stale"
	DataQualityUncertain DataQuality = "uncertain"
	DataQualityInvalid   DataQuality = "invalid"
)

// GapClass buckets elapsed time between samples.
type GapClass string

const (
	GapNormal   GapClass = "normal"
	GapLong     GapClass = "long"
	GapTooLarge GapClass = "too_large"
)

// Pending tracks hysteresis for phase transitions.
type Pending struct {
	Phase Phase     `json:"phase"`
	Count int       `json:"count"`
	Since time.Time `json:"since"`
}

// APPlus holds autopilot-plus resumption hypothesis fields.
type APPlus struct {
	Candidate bool      `json:"candidate"`
	Score     float64   `json:"score"`
	Since     time.Time `json:"since"`
	Reasons   []string  `json:"reasons,omitempty"`
}

// WindowFeatureSnapshot is a compact diagnostic copy of window features.
type WindowFeatureSnapshot struct {
	ElapsedSec            float64  `json:"elapsedSec,omitempty"`
	DistanceNM            float64  `json:"distanceNm,omitempty"`
	AltDeltaFt            float64  `json:"altDeltaFt,omitempty"`
	AvgVerticalFpm        float64  `json:"avgVerticalFpm,omitempty"`
	SpeedDeltaKt          float64  `json:"speedDeltaKt,omitempty"`
	TrackDeltaDeg         float64  `json:"trackDeltaDeg,omitempty"`
	Moved                 bool     `json:"moved"`
	AltStable             bool     `json:"altStable"`
	ProgressRatio         float64  `json:"progressRatio,omitempty"`
	GapClass              GapClass `json:"gapClass,omitempty"`
	Fresh                 bool     `json:"fresh"`
	StationarySamples     int      `json:"stationarySamples"`
	StationaryDurationSec float64  `json:"stationaryDurationSec,omitempty"`
	SustainedClimbCount   int      `json:"sustainedClimbCount"`
	SustainedDescentCount int      `json:"sustainedDescentCount"`
	LastReportAdvancing   bool     `json:"lastReportAdvancing"`
}

// State is the full estimator state kept in Redis history.
type State struct {
	Phase          Phase                 `json:"phase"`
	Activity       Activity              `json:"activity"`
	DataQuality    DataQuality           `json:"dataQuality"`
	Confidence     float64               `json:"confidence"`
	PreviousPhase  Phase                 `json:"previousPhase,omitempty"`
	PhaseSince     time.Time             `json:"phaseSince"`
	Pending        *Pending              `json:"pending,omitempty"`
	GroundRefAltFt float64               `json:"groundRefAltFt,omitempty"`
	Reasons        []string              `json:"reasons,omitempty"`
	APPlus         APPlus                `json:"apPlus"`
	Features       WindowFeatureSnapshot `json:"features,omitempty"`
}
