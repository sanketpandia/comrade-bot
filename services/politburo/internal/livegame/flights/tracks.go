package flights

// Track is a motion-oriented row returned by the active flights API.
type Track struct {
	FlightID  string
	Callsign  string
	Latitude  float64
	Longitude float64
	Speed     int
	Track     float64
}

func TrackFromMotion(flightID string, motion FlightMotion) Track {
	return Track{
		FlightID:  flightID,
		Callsign:  motion.Callsign,
		Latitude:  motion.Latitude,
		Longitude: motion.Longitude,
		Speed:     motion.Speed,
		Track:     motion.Track,
	}
}

func TrackFromFlight(flight Flight) Track {
	return Track{
		FlightID:  flight.FlightID,
		Callsign:  flight.Callsign,
		Latitude:  flight.Latitude,
		Longitude: flight.Longitude,
		Speed:     flight.Speed,
		Track:     flight.Track,
	}
}
