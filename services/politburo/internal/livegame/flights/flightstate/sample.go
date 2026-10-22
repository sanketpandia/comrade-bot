package flightstate

import (
	"math"
	"time"
)

// Sample is one observation tick with explicit validity for nullable measurements.
type Sample struct {
	LastReport      time.Time `json:"lastReport"`
	LastReportValid bool      `json:"lastReportValid"`
	ObservedAt      time.Time `json:"observedAt"`

	LatitudeValid  bool    `json:"latitudeValid"`
	LongitudeValid bool    `json:"longitudeValid"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`

	AltitudeValid bool    `json:"altitudeValid"`
	Altitude      float64 `json:"altitude"`

	SpeedValid bool    `json:"speedValid"`
	Speed      float64 `json:"speed"`

	VerticalSpeedValid bool    `json:"verticalSpeedValid"`
	VerticalSpeed      float64 `json:"verticalSpeed"`

	TrackValid bool    `json:"trackValid"`
	Track      float64 `json:"track"`

	IsConnected bool `json:"isConnected"`
	PilotState  int  `json:"pilotState"`
}

// PilotStateInBackground is Infinite Flight's pilotState when the app is backgrounded (AP+).
const PilotStateInBackground = 3

func validLatitude(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -90 && v <= 90
}

func validLongitude(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -180 && v <= 180
}

func (s Sample) coordsValid() bool {
	return s.LatitudeValid && s.LongitudeValid
}

func (s Sample) coreValid() bool {
	return s.coordsValid() && s.AltitudeValid && s.SpeedValid && s.LastReportValid
}
