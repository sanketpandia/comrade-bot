package virtualairlines

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
)

const webhookDispatchQuery = `
SELECT va.id, va.code, wh.config, basic.config
FROM public.va_configs wh
INNER JOIN public.virtual_airlines va ON va.id = wh.va_id
LEFT JOIN public.va_configs basic
    ON basic.va_id = va.id AND basic.config_type = $2
WHERE wh.config_type = $1 AND va.is_active = true
ORDER BY va.code`

// WebhookDispatchRow is one active VA that has a WEBHOOKS config.
// PreferredServers comes from that VA's BASIC config and is empty when BASIC is missing.
type WebhookDispatchRow struct {
	VAID             string
	VACode           string
	Webhooks         json.RawMessage
	PreferredServers []string
}

// LoadWebhookDispatchRows returns WEBHOOKS documents for active VAs plus BASIC preferred servers.
func (s *Service) LoadWebhookDispatchRows(ctx context.Context) ([]WebhookDispatchRow, error) {
	rows, err := s.db.QueryContext(ctx, webhookDispatchQuery, vaconfig.TypeWebhooks, vaconfig.TypeBasic)
	if err != nil {
		return nil, fmt.Errorf("query webhook va configs: %w", err)
	}
	defer rows.Close()

	var out []WebhookDispatchRow
	for rows.Next() {
		var vaID, vaCode string
		var webhooks, basic []byte
		if err := rows.Scan(&vaID, &vaCode, &webhooks, &basic); err != nil {
			return nil, fmt.Errorf("scan webhook va config row: %w", err)
		}
		row := WebhookDispatchRow{
			VAID:     vaID,
			VACode:   vaCode,
			Webhooks: json.RawMessage(webhooks),
		}
		if len(basic) > 0 {
			servers, err := preferredServersFromBasic(basic)
			if err != nil {
				slog.Warn("skip preferred servers on invalid BASIC config", "va_id", vaID, "error", err)
			} else {
				row.PreferredServers = servers
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhook va configs: %w", err)
	}
	return out, nil
}

func preferredServersFromBasic(config []byte) ([]string, error) {
	row, err := parseBasicConfigRow("", "", config)
	if err != nil {
		return nil, err
	}
	return row.PreferredServers, nil
}
