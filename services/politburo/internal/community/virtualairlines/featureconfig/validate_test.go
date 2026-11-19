package featureconfig

import (
	"testing"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
)

func TestValidateStatsDraftRequiresBindings(t *testing.T) {
	doc := StatsDocument{
		Enabled: true,
		Pilot:   &StatsSection{MatchBy: "callsign", Fields: []string{"displayName"}},
	}
	err := ValidateStatsDraft(doc, map[string]connections.Mapping{})
	if err == nil {
		t.Fatal("expected error for missing pilots mapping")
	}
}

func TestValidateCareerIFCUsernameViaRoster(t *testing.T) {
	doc := StatsDocument{
		Enabled: true,
		Career:  &StatsSection{MatchBy: "ifcUsername", Fields: []string{"flightTimeSeconds"}},
	}
	mappings := map[string]connections.Mapping{
		connections.EntityPilots: {
			TableID: "tblP",
			Fields: map[string]string{
				"ifcUsername": "fldIfc",
				"callsign":    "fldCs",
			},
		},
		connections.EntityCareerPilots: {
			TableID: "tblC",
			Fields: map[string]string{
				"callsign":           "fldCsCareer",
				"flightTimeSeconds": "fldHours",
			},
		},
	}
	if err := ValidateStatsDraft(doc, mappings); err != nil {
		t.Fatalf("expected indirect career ifc match: %v", err)
	}
}

func TestValidatePIREPDraftRequiresPilotField(t *testing.T) {
	doc := PIREPDocument{Enabled: true, Fields: []string{"remarks"}}
	mappings := map[string]connections.Mapping{
		connections.EntityPIREPs: {
			TableID: "tblPireps",
			Fields:  map[string]string{"remarks": "fldRemarks"},
		},
	}
	err := ValidatePIREPDraft(doc, mappings)
	if err == nil {
		t.Fatal("expected error without pilot reference")
	}
}