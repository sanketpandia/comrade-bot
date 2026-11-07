package infiniteflight

import "testing"

func TestScheduledFlightStatusRoundTrip(t *testing.T) {
	for _, want := range []struct {
		code int
		name string
	}{
		{0, "Unknown"},
		{1, "Scheduled"},
		{6, "InFlight"},
		{11, "Arrived"},
	} {
		got, ok := ScheduledFlightStatusName(want.code)
		if !ok || got != want.name {
			t.Fatalf("ScheduledFlightStatusName(%d) = %q, %v", want.code, got, ok)
		}
		code, err := ParseScheduledFlightStatus(want.name)
		if err != nil || code != want.code {
			t.Fatalf("ParseScheduledFlightStatus(%q) = %d, %v", want.name, code, err)
		}
	}
}

func TestPersistentFlightTypeRoundTrip(t *testing.T) {
	name, ok := PersistentFlightTypeName(1)
	if !ok || name != "Commercial" {
		t.Fatalf("PersistentFlightTypeName(1) = %q, %v", name, ok)
	}
	code, err := ParsePersistentFlightType("Commercial")
	if err != nil || code != 1 {
		t.Fatalf("ParsePersistentFlightType: %d, %v", code, err)
	}
}

func TestScheduleEnumOptionsSkipsUnusedStatusSlot(t *testing.T) {
	opts := ScheduleEnumOptions()
	for _, name := range opts.Status {
		if name == "" {
			t.Fatal("status list contains empty name")
		}
	}
	if len(opts.FlightType) != 13 {
		t.Fatalf("flight type count = %d", len(opts.FlightType))
	}
}

func TestPersistentFlightCrewMemberRoleRoundTrip(t *testing.T) {
	name, ok := PersistentFlightCrewMemberRoleName(1)
	if !ok || name != "FirstOfficer" {
		t.Fatalf("PersistentFlightCrewMemberRoleName(1) = %q, %v", name, ok)
	}
	code, err := ParsePersistentFlightCrewMemberRole("Captain")
	if err != nil || code != 0 {
		t.Fatalf("ParsePersistentFlightCrewMemberRole: %d, %v", code, err)
	}
}

func TestScheduleEnumOptionsIncludesCrewRole(t *testing.T) {
	opts := ScheduleEnumOptions()
	if len(opts.CrewRole) != 2 || opts.CrewRole[0] != "Captain" {
		t.Fatalf("crewRole = %+v", opts.CrewRole)
	}
}
