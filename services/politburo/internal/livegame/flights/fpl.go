package flights

import "time"

// FPLSyncDue reports whether an active pilot is due for flight-plan sync on this tick.
func FPLSyncDue(pilotState int, fpl FPLSyncAction, now time.Time) bool {
	if pilotState != PilotStateActive {
		return false
	}
	if fpl.LastAt.IsZero() {
		return true
	}
	return now.Sub(fpl.LastAt) >= FPLSyncInterval
}
