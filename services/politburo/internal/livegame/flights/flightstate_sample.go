package flights

import (
	"math"

	"infinite-experiment/politburo/internal/livegame/flights/flightstate"
)

// FlightStateSample maps an observation into a temporal estimator sample.
func FlightStateSample(obs Observation) flightstate.Sample {
	s := flightstate.Sample{
		LastReport:      obs.LastReport,
		LastReportValid: obs.LastReportValid,
		ObservedAt:      obs.ObservedAt,
		IsConnected:     obs.IsConnected,
		PilotState:      obs.PilotState,
	}
	if validFlightStateLatitude(obs.Latitude) {
		s.LatitudeValid = true
		s.Latitude = obs.Latitude
	}
	if validFlightStateLongitude(obs.Longitude) {
		s.LongitudeValid = true
		s.Longitude = obs.Longitude
	}
	if !math.IsNaN(obs.Altitude) && !math.IsInf(obs.Altitude, 0) {
		s.AltitudeValid = true
		s.Altitude = obs.Altitude
	}
	if !math.IsNaN(obs.Speed) && !math.IsInf(obs.Speed, 0) && obs.Speed >= 0 {
		s.SpeedValid = true
		s.Speed = obs.Speed
	}
	if !math.IsNaN(obs.VerticalSpeed) && !math.IsInf(obs.VerticalSpeed, 0) {
		s.VerticalSpeedValid = true
		s.VerticalSpeed = obs.VerticalSpeed
	}
	if !math.IsNaN(obs.Track) && !math.IsInf(obs.Track, 0) {
		s.TrackValid = true
		s.Track = obs.Track
	}
	return s
}

func validFlightStateLatitude(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -90 && v <= 90
}

func validFlightStateLongitude(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -180 && v <= 180
}
