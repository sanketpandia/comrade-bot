package virtualairlines

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	vaconfig "infinite-experiment/politburo/internal/community/virtualairlines/config"
	webhookconfig "infinite-experiment/politburo/internal/webhooks/config"
)

var (
	ErrWebhookConfigNotFound = errors.New("webhooks config not found")
	ErrWebhookTypeNotFound   = errors.New("webhook type not found")
)

const webhookDispatchQuery = `
SELECT va.id, va.code, wh.config, basic.config
FROM public.va_configs wh
INNER JOIN public.virtual_airlines va ON va.id = wh.va_id
LEFT JOIN public.va_configs basic
    ON basic.va_id = va.id AND basic.config_type = $2
WHERE wh.config_type = $1 AND va.is_active = true
ORDER BY va.code`

const webhookDispatchTargetQuery = `
SELECT va.id, va.code, wh.config, basic.config
FROM public.virtual_airlines va
INNER JOIN public.va_configs wh ON wh.va_id = va.id AND wh.config_type = $2
LEFT JOIN public.va_configs basic
    ON basic.va_id = va.id AND basic.config_type = $3
WHERE va.id = $1 AND va.is_active = true`

// WebhookDispatchTarget is one WEBHOOKS feature entry for manual or scheduled dispatch.
type WebhookDispatchTarget struct {
	VAID             string
	VACode           string
	WebhookType      string
	URL              string
	Filters          json.RawMessage
	PreferredServers []string
}

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

// LoadWebhookDispatchTarget loads one feature entry from an active VA's WEBHOOKS config.
func (s *Service) LoadWebhookDispatchTarget(ctx context.Context, vaID, webhookType string) (WebhookDispatchTarget, error) {
	webhookType = strings.TrimSpace(webhookType)
	if webhookType == "" {
		return WebhookDispatchTarget{}, ErrWebhookTypeNotFound
	}

	var id, vaCode string
	var webhooks, basic []byte
	err := s.db.QueryRowContext(ctx, webhookDispatchTargetQuery, vaID, vaconfig.TypeWebhooks, vaconfig.TypeBasic).
		Scan(&id, &vaCode, &webhooks, &basic)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WebhookDispatchTarget{}, ErrWebhookConfigNotFound
		}
		return WebhookDispatchTarget{}, fmt.Errorf("query webhook dispatch target: %w", err)
	}

	entries, err := webhookconfig.Parse(json.RawMessage(webhooks))
	if err != nil {
		return WebhookDispatchTarget{}, fmt.Errorf("parse webhooks config: %w", err)
	}
	for _, entry := range entries {
		if entry.Key != webhookType {
			continue
		}
		target := WebhookDispatchTarget{
			VAID:        id,
			VACode:      vaCode,
			WebhookType: entry.Key,
			URL:         entry.URL,
			Filters:     entry.Filters,
		}
		if len(basic) > 0 {
			servers, err := preferredServersFromBasic(basic)
			if err != nil {
				slog.Warn("skip preferred servers on invalid BASIC config", "va_id", vaID, "error", err)
			} else {
				target.PreferredServers = servers
			}
		}
		return target, nil
	}
	return WebhookDispatchTarget{}, ErrWebhookTypeNotFound
}

func preferredServersFromBasic(config []byte) ([]string, error) {
	row, err := parseBasicConfigRow("", "", config)
	if err != nil {
		return nil, err
	}
	return row.PreferredServers, nil
}
