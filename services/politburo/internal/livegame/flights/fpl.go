package flights

import "time"

// PlanFresh reports whether a cached plan fetchedAt is younger than FPLSyncInterval.
func PlanFresh(fetchedAt, now time.Time) bool {
	if fetchedAt.IsZero() {
		return false
	}
	return now.Sub(fetchedAt) < FPLSyncInterval
}

// PlanSyncDue reports whether a VA-linked active pilot needs a background plan fetch.
func PlanSyncDue(vaLinked bool, pilotState int, planFetchedAt, now time.Time) bool {
	if !vaLinked || pilotState != PilotStateActive {
		return false
	}
	return !PlanFresh(planFetchedAt, now)
}
