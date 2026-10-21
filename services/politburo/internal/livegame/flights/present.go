package flights

import "math"

// Presentation helpers round raw observation values for API and motion snapshots.

func RoundedLatitude(lat float64) float64 {
	return math.Round(lat*10000) / 10000
}

func RoundedLongitude(lon float64) float64 {
	return math.Round(lon*10000) / 10000
}

func RoundedAltitude(alt float64) int {
	return int(math.Round(alt))
}

func RoundedSpeed(speed float64) int {
	return int(math.Round(speed))
}

func RoundedVerticalSpeed(vs float64) float64 {
	return math.Round(vs*10) / 10
}

func RoundedTrack(track float64) float64 {
	return math.Round(track*10) / 10
}
