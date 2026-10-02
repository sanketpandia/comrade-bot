package flights

import "testing"

func TestShouldResolveLivery(t *testing.T) {
	prior := &Flight{AircraftID: "a1", LiveryID: "l1", AircraftName: "A320"}
	if ShouldResolveLivery(prior, "a1", "l1") {
		t.Fatal("expected skip when names already resolved")
	}
	if !ShouldResolveLivery(prior, "a2", "l1") {
		t.Fatal("expected resolve when aircraft changed")
	}
	if !ShouldResolveLivery(nil, "a1", "l1") {
		t.Fatal("expected resolve without prior")
	}
}
