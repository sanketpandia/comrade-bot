package virtualairlines

import (
	"context"
	"fmt"
	"log/slog"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
)

// LoadLiveFlightVACatalog loads all active VA BASIC configs and builds the live-flight catalog.
func (s *Service) LoadLiveFlightVACatalog(ctx context.Context) (LiveVACatalog, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT va.id, va.code, vc.config
FROM public.va_configs vc
INNER JOIN public.virtual_airlines va ON va.id = vc.va_id
WHERE vc.config_type = $1 AND va.is_active = true`,
		vaconfig.TypeBasic,
	)
	if err != nil {
		return LiveVACatalog{}, fmt.Errorf("query basic va configs: %w", err)
	}
	defer rows.Close()

	var parsed []basicConfigRow
	for rows.Next() {
		var vaID, vaCode string
		var config []byte
		if err := rows.Scan(&vaID, &vaCode, &config); err != nil {
			return LiveVACatalog{}, fmt.Errorf("scan basic va config row: %w", err)
		}
		row, err := parseBasicConfigRow(vaID, vaCode, config)
		if err != nil {
			slog.Warn("skip invalid BASIC va config row", "vaId", vaID, "error", err)
			continue
		}
		parsed = append(parsed, row)
	}
	if err := rows.Err(); err != nil {
		return LiveVACatalog{}, fmt.Errorf("iterate basic va configs: %w", err)
	}
	return BuildLiveVACatalog(parsed), nil
}
