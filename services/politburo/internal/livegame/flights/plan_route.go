package flights

import "strings"

// PlanRouteEndpoints returns departure and destination labels from the first and
// last cached plan waypoints.
func PlanRouteEndpoints(plan CachedFlightPlan) (departure, destination string, ok bool) {
	if plan.Absent || len(plan.Waypoints) == 0 {
		return "", "", false
	}
	departure = planWaypointLabel(plan.Waypoints[0], 1)
	destination = planWaypointLabel(plan.Waypoints[len(plan.Waypoints)-1], len(plan.Waypoints))
	return departure, destination, departure != "" || destination != ""
}

func planWaypointLabel(wp PlanWaypoint, seq int) string {
	if name := strings.TrimSpace(wp.Name); name != "" {
		return name
	}
	return ""
}
