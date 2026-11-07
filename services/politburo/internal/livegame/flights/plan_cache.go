package flights

import (
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

// PlanWaypoint is one drawn point from a cached flight plan.
type PlanWaypoint struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  int     `json:"altitude"`
}

// CachedFlightPlan is the Redis value at game:flights:plan:<flightId>.
type CachedFlightPlan struct {
	FetchedAt      time.Time      `json:"fetchedAt"`
	FlightPlanID   string         `json:"flightPlanId,omitempty"`
	FlightPlanType *int           `json:"flightPlanType,omitempty"`
	Waypoints      []PlanWaypoint `json:"waypoints,omitempty"`
	Absent         bool           `json:"absent,omitempty"`
}

// FlattenFlightPlanWaypoints walks flightPlanItems depth-first with coordinates.
func FlattenFlightPlanWaypoints(plan infiniteflight.FlightPlan) []PlanWaypoint {
	out := make([]PlanWaypoint, 0, len(plan.Items))
	var walk func([]infiniteflight.FlightPlanItem)
	walk = func(items []infiniteflight.FlightPlanItem) {
		for _, item := range items {
			if item.Location.Latitude != 0 || item.Location.Longitude != 0 {
				out = append(out, PlanWaypoint{
					Name:      item.Name,
					Latitude:  item.Location.Latitude,
					Longitude: item.Location.Longitude,
					Altitude:  item.Altitude,
				})
			}
			if len(item.Children) > 0 {
				walk(item.Children)
			}
		}
	}
	walk(plan.Items)
	return out
}

// FPLDueEntry is one row in the background plan-sync due list.
type FPLDueEntry struct {
	SessionID string `json:"sessionId"`
	FlightID  string `json:"flightId"`
}
