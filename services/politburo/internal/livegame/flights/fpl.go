package flights

import "time"

// FPLSyncDue reports whether an active pilot is due for flight-plan sync on this tick.
func FPLSyncDue(pilotState int, pathSync *PathSync, now time.Time) bool {
	if pilotState != PilotStateActive {
		return false
	}
	if pathSync == nil || pathSync.LastFPLSyncAt.IsZero() {
		return true
	}
	return now.Sub(pathSync.LastFPLSyncAt) >= FPLSyncInterval
}
