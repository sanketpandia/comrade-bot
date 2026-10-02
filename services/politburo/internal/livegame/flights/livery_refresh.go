package flights

// ShouldResolveLivery reports whether the livery catalog should be consulted for this tick.
func ShouldResolveLivery(prior *Flight, aircraftID, liveryID string) bool {
	if prior == nil {
		return true
	}
	if prior.AircraftID != aircraftID || prior.LiveryID != liveryID {
		return true
	}
	return prior.AircraftName == ""
}
