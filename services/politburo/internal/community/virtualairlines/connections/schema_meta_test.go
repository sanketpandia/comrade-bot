package connections

import (
	"testing"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

func TestTypeCompatibleLastModifiedTimeAsLastActivity(t *testing.T) {
	cf, ok := catalogField(EntityCareerPilots, "lastActivity")
	if !ok {
		t.Fatal("missing lastActivity catalog field")
	}
	field := airtable.Field{
		ID:   "fld1",
		Name: "Last activity in CM",
		Type: "lastModifiedTime",
	}
	if !typeCompatible(cf, field) {
		t.Fatal("expected lastModifiedTime to satisfy lastActivity")
	}
	if got := effectiveAirtableType(field); got != "dateTime" {
		t.Fatalf("effective type = %q, want dateTime", got)
	}
}