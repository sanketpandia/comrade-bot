package connections

import (
	"fmt"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

// FilterOpts tweaks equality filter construction.
type FilterOpts struct {
	CallsignMatch bool
}

// NormalizeCallsignForMatch uppercases and removes spaces for roster equality.
func NormalizeCallsignForMatch(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

// BuildEqualityFilter returns one Airtable filterByFormula equality clause.
func BuildEqualityFilter(meta FieldMeta, value any, opts FilterOpts) (string, error) {
	lit, err := filterLiteralValue(value)
	if err != nil {
		return "", err
	}
	ref := airtable.FieldRef(meta.Name)
	if opts.CallsignMatch {
		norm := NormalizeCallsignForMatch(lit)
		return fmt.Sprintf("UPPER(SUBSTITUTE(%s,\" \",\"\"))='%s'", ref, airtable.EscapeFormulaLiteral(norm)), nil
	}
	return fmt.Sprintf("%s='%s'", ref, airtable.EscapeFormulaLiteral(lit)), nil
}