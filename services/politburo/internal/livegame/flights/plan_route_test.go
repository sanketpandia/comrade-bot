package flights

import "testing"

func TestPlanRouteEndpoints(t *testing.T) {
	dep, dest, ok := PlanRouteEndpoints(CachedFlightPlan{
		Waypoints: []PlanWaypoint{
			{Name: " EGLL "},
			{Name: "BOBAG"},
			{Name: "KJFK"},
		},
	})
	if !ok || dep != "EGLL" || dest != "KJFK" {
		t.Fatalf("got %q %q ok=%v", dep, dest, ok)
	}

	_, _, ok = PlanRouteEndpoints(CachedFlightPlan{Absent: true})
	if ok {
		t.Fatal("expected absent plan to yield no endpoints")
	}
}
