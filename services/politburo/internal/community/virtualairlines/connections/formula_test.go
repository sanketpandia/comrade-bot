package connections

import (
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

func TestEscapeFormulaLiteral(t *testing.T) {
	cases := map[string]string{
		`plain`:    `plain`,
		`O'Brien`:  `O\'Brien`,
		`back\`:    `back\\`,
		`\'`:       `\\\'`,
	}
	for in, want := range cases {
		if got := airtable.EscapeFormulaLiteral(in); got != want {
			t.Fatalf("%q => %q want %q", in, got, want)
		}
	}
}

func TestBuildEqualityFilterCallsign(t *testing.T) {
	meta := FieldMeta{Name: "Pilot ID"}
	clause, err := BuildEqualityFilter(meta, "nx 042", FilterOpts{CallsignMatch: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(clause, "UPPER(SUBSTITUTE") || !strings.Contains(clause, "NX042") {
		t.Fatalf("clause = %s", clause)
	}
}

func TestFieldRefEscapesBraces(t *testing.T) {
	ref := airtable.FieldRef("Col}Name")
	if ref != "{Col\\}Name}" {
		t.Fatalf("ref = %s", ref)
	}
}