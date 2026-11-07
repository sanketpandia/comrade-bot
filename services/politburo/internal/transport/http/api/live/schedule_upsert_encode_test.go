package live

import (
	"encoding/json"
	"testing"

	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
)

func TestEncodeScheduleUpsertForUpstreamUsesInts(t *testing.T) {
	status := politburoapi.LiveScheduleStatusName("Scheduled")
	ft := politburoapi.LiveScheduleFlightTypeName("Commercial")
	raw, err := encodeScheduleUpsertForUpstream(politburoapi.LiveScheduleUpsertBody{
		Status:     &status,
		FlightType: &ft,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var upstream map[string]any
	if err := json.Unmarshal(raw, &upstream); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if upstream["status"] != float64(1) {
		t.Fatalf("status = %v", upstream["status"])
	}
	if upstream["flightType"] != float64(1) {
		t.Fatalf("flightType = %v", upstream["flightType"])
	}
}
