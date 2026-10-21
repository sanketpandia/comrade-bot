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

func TrackFromRecord(record Record) Track {
	raw := record.Raw
	return Track{
		FlightID:  raw.FlightID,
		Callsign:  raw.Callsign,
		Latitude:  RoundedLatitude(raw.Latitude),
		Longitude: RoundedLongitude(raw.Longitude),
		Speed:     RoundedSpeed(raw.Speed),
		Track:     RoundedTrack(raw.Track),
	}
}
