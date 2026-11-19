package connections

import (
	"strings"
	"testing"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

func TestSummarizeTables(t *testing.T) {
	tables := []airtable.Table{
		{Name: "A", ID: "tblA"},
		{Name: "B", ID: "tblB"},
		{Name: "C", ID: "tblC"},
	}
	got := summarizeTables(tables, 2)
	if !strings.Contains(got, `"A" (tblA)`) || !strings.Contains(got, "1 more") {
		t.Fatalf("unexpected summary: %q", got)
	}
}