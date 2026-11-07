package infiniteflight

import (
	"fmt"
	"strings"
)

// Scheduled flight status names match Infinite Flight Live API ScheduledFlightStatus (no value 5).
var scheduledFlightStatusNames = []string{
	"Unknown",
	"Scheduled",
	"Boarding",
	"Boarded",
	"TaxiingToRunway",
	"", // 5 — unused upstream
	"InFlight",
	"Diverted",
	"Delayed",
	"Cancelled",
	"TaxiingToParking",
	"Arrived",
}

// Persistent flight type names match Infinite Flight Live API PersistentFlightType.
var persistentFlightTypeNames = []string{
	"None",
	"Commercial",
	"Charter",
	"Cargo",
	"Training",
	"TestFlight",
	"MedicalEmergency",
	"Military",
	"VIPExecutive",
	"HumanitarianRelief",
	"GeneralAviation",
	"Airshow",
	"Other",
}

// Persistent flight crew role names match Infinite Flight Live API PersistentFlightCrewMemberRole.
var persistentFlightCrewMemberRoleNames = []string{
	"Captain",
	"FirstOfficer",
}

// ScheduledFlightStatusName returns the API name for an upstream schedule status code.
func ScheduledFlightStatusName(code int) (string, bool) {
	if code < 0 || code >= len(scheduledFlightStatusNames) {
		return "", false
	}
	name := scheduledFlightStatusNames[code]
	if name == "" {
		return "", false
	}
	return name, true
}

// ParseScheduledFlightStatus maps an API status name to the upstream integer.
func ParseScheduledFlightStatus(name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("schedule status is empty")
	}
	for i, candidate := range scheduledFlightStatusNames {
		if candidate != "" && candidate == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown schedule status %q", name)
}

// PersistentFlightTypeName returns the API name for an upstream flight type code.
func PersistentFlightTypeName(code int) (string, bool) {
	if code < 0 || code >= len(persistentFlightTypeNames) {
		return "", false
	}
	return persistentFlightTypeNames[code], true
}

// ParsePersistentFlightType maps an API flight type name to the upstream integer.
func ParsePersistentFlightType(name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("flight type is empty")
	}
	for i, candidate := range persistentFlightTypeNames {
		if candidate == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown flight type %q", name)
}

// PersistentFlightCrewMemberRoleName returns the API name for an upstream crew role code.
func PersistentFlightCrewMemberRoleName(code int) (string, bool) {
	if code < 0 || code >= len(persistentFlightCrewMemberRoleNames) {
		return "", false
	}
	return persistentFlightCrewMemberRoleNames[code], true
}

// ParsePersistentFlightCrewMemberRole maps an API crew role name to the upstream integer.
func ParsePersistentFlightCrewMemberRole(name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("crew role is empty")
	}
	for i, candidate := range persistentFlightCrewMemberRoleNames {
		if candidate == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown crew role %q", name)
}

// LiveScheduleEnumOptions is the catalog of string enum values exposed to API clients.
type LiveScheduleEnumOptions struct {
	Status     []string
	FlightType []string
	CrewRole   []string
}

// ScheduleEnumOptions returns all valid schedule-related API enum names.
func ScheduleEnumOptions() LiveScheduleEnumOptions {
	status := make([]string, 0, len(scheduledFlightStatusNames))
	for _, name := range scheduledFlightStatusNames {
		if name != "" {
			status = append(status, name)
		}
	}
	flightType := make([]string, len(persistentFlightTypeNames))
	copy(flightType, persistentFlightTypeNames)
	crewRole := make([]string, len(persistentFlightCrewMemberRoleNames))
	copy(crewRole, persistentFlightCrewMemberRoleNames)
	return LiveScheduleEnumOptions{Status: status, FlightType: flightType, CrewRole: crewRole}
}
