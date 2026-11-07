package valive

import (
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func TestMatchAircraftIDsFromRowsCaseFold(t *testing.T) {
	rows := []infiniteflight.LiveAircraft{
		{ID: "row-1", Registration: "ph-znb"},
		{ID: "row-2", Registration: "OTHER"},
	}
	ids, regs := matchAircraftIDsFromRows(rows, "PH-ZNB")
	if len(ids) != 1 || ids[0] != "row-1" {
		t.Fatalf("ids = %v", ids)
	}
	if regs["row-1"] != "ph-znb" {
		t.Fatalf("regs = %v", regs)
	}
}

func TestMatchAircraftIDsFromRowsEmpty(t *testing.T) {
	ids, _ := matchAircraftIDsFromRows(nil, "PH-ZNB")
	if len(ids) != 0 {
		t.Fatalf("ids = %v", ids)
	}
}
