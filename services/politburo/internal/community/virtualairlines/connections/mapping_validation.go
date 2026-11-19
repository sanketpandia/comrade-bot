package connections

import (
	"fmt"
	"log/slog"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections/airtable"
)

func logMappingSaveRejected(vaID, entity, kind string, attrs ...any) {
	args := []any{
		"component", "data_connections",
		"va_id", vaID,
		"entity", entity,
		"operation", "save_mapping",
		"error_kind", kind,
	}
	args = append(args, attrs...)
	slog.Warn("data connection mapping save rejected", args...)
}

func summarizeTables(tables []airtable.Table, limit int) string {
	if len(tables) == 0 {
		return "(none)"
	}
	if limit <= 0 || limit > len(tables) {
		limit = len(tables)
	}
	parts := make([]string, 0, limit+1)
	for i := 0; i < limit; i++ {
		t := tables[i]
		parts = append(parts, fmt.Sprintf("%q (%s)", t.Name, t.ID))
	}
	out := strings.Join(parts, ", ")
	if limit < len(tables) {
		out += fmt.Sprintf(", … (%d more)", len(tables)-limit)
	}
	return out
}