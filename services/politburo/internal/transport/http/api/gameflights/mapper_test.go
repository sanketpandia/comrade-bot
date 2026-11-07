package gameflights

import (
	"testing"

	"github.com/google/uuid"

	domainflights "infinite-experiment/politburo/internal/livegame/flights"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestActiveFlightRecordFromDomainIncludesUsers(t *testing.T) {
	flightID := uuid.MustParse("c34118e7-cbdd-4e22-8751-0cda93e41d75")
	userA := "daa083c0-8bc9-4750-ae37-3cec5695841c"
	record := domainflights.Record{
		Raw: domainflights.Observation{
			FlightID: flightID.String(),
			Callsign: "Swiss 39 Heavy",
			UserID:   userA,
		},
		Links: domainflights.Links{
			Users: []string{userA},
		},
	}

	apiRecord := activeFlightRecordFromDomain(record, openapi_types.UUID(flightID), nil)
	if apiRecord.Links.Users == nil || len(*apiRecord.Links.Users) != 1 || (*apiRecord.Links.Users)[0] != userA {
		t.Fatalf("users = %#v", apiRecord.Links.Users)
	}
}
