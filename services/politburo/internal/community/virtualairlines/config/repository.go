package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const blockColumns = `va_id, config_type, config, updated_at, updated_by`

func (r *Repository) Get(ctx context.Context, vaID, configType string) (*Block, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT `+blockColumns+`
FROM public.va_configs
WHERE va_id = $1 AND config_type = $2`,
		vaID, configType,
	)
	block, err := scanBlock(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get va config block: %w", err)
	}
	return block, nil
}

func (r *Repository) Upsert(ctx context.Context, vaID, configType string, config json.RawMessage, updatedBy string) (*Block, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.va_configs (va_id, config_type, config, updated_at, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (va_id, config_type) DO UPDATE SET
    config = EXCLUDED.config,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by
RETURNING `+blockColumns,
		vaID, configType, []byte(config), time.Now().UTC(), updatedBy,
	)
	block, err := scanBlock(row)
	if err != nil {
		return nil, fmt.Errorf("upsert va config block: %w", err)
	}
	return block, nil
}

// ActiveBasic is an active VA's parsed BASIC block.
type ActiveBasic struct {
	VAID   string
	VACode string
	Basic  Basic
}

// ActiveWebhooks is an active VA's raw WEBHOOKS block. PreferredServers comes from the
// VA's BASIC block and is empty when BASIC is missing or invalid.
type ActiveWebhooks struct {
	VAID             string
	VACode           string
	Webhooks         json.RawMessage
	PreferredServers []string
}

const listActiveBasicQuery = `
SELECT va.id, va.code, vc.config
FROM public.va_configs vc
INNER JOIN public.virtual_airlines va ON va.id = vc.va_id
WHERE vc.config_type = $1 AND va.is_active = true
ORDER BY va.code`

const activeWebhooksColumns = `
SELECT va.id, va.code, wh.config, basic.config
FROM public.va_configs wh
INNER JOIN public.virtual_airlines va ON va.id = wh.va_id
LEFT JOIN public.va_configs basic ON basic.va_id = va.id AND basic.config_type = $2
WHERE wh.config_type = $1 AND va.is_active = true`

const listActiveWebhooksQuery = activeWebhooksColumns + `
ORDER BY va.code`

const getActiveWebhooksQuery = activeWebhooksColumns + `
  AND va.id = $3`

// ListActiveBasic returns parsed BASIC blocks for active VAs. Rows that fail to parse are logged and skipped.
func (r *Repository) ListActiveBasic(ctx context.Context) ([]ActiveBasic, error) {
	rows, err := r.db.QueryContext(ctx, listActiveBasicQuery, TypeBasic)
	if err != nil {
		return nil, fmt.Errorf("list active basic configs: %w", err)
	}
	defer rows.Close()

	var out []ActiveBasic
	for rows.Next() {
		var row ActiveBasic
		var raw []byte
		if err := rows.Scan(&row.VAID, &row.VACode, &raw); err != nil {
			return nil, fmt.Errorf("scan active basic config: %w", err)
		}
		basic, err := ParseBasic(raw)
		if err != nil {
			slog.Warn("skip invalid BASIC va config", "va_id", row.VAID, "error", err)
			continue
		}
		row.Basic = basic
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active basic configs: %w", err)
	}
	return out, nil
}

// ListActiveWebhooks returns WEBHOOKS blocks for active VAs, ordered by VA code.
func (r *Repository) ListActiveWebhooks(ctx context.Context) ([]ActiveWebhooks, error) {
	rows, err := r.db.QueryContext(ctx, listActiveWebhooksQuery, TypeWebhooks, TypeBasic)
	if err != nil {
		return nil, fmt.Errorf("list active webhooks configs: %w", err)
	}
	defer rows.Close()

	var out []ActiveWebhooks
	for rows.Next() {
		row, err := scanActiveWebhooks(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active webhooks config: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active webhooks configs: %w", err)
	}
	return out, nil
}

// GetActiveWebhooks returns one active VA's WEBHOOKS block, or ErrNotFound.
func (r *Repository) GetActiveWebhooks(ctx context.Context, vaID string) (ActiveWebhooks, error) {
	row, err := scanActiveWebhooks(r.db.QueryRowContext(ctx, getActiveWebhooksQuery, TypeWebhooks, TypeBasic, vaID))
	if errors.Is(err, sql.ErrNoRows) {
		return ActiveWebhooks{}, ErrNotFound
	}
	if err != nil {
		return ActiveWebhooks{}, fmt.Errorf("get active webhooks config: %w", err)
	}
	return row, nil
}

func scanActiveWebhooks(row interface{ Scan(dest ...any) error }) (ActiveWebhooks, error) {
	var out ActiveWebhooks
	var webhooks, basicRaw []byte
	if err := row.Scan(&out.VAID, &out.VACode, &webhooks, &basicRaw); err != nil {
		return ActiveWebhooks{}, err
	}
	out.Webhooks = json.RawMessage(webhooks)
	if len(basicRaw) > 0 {
		basic, err := ParseBasic(basicRaw)
		if err != nil {
			slog.Warn("ignore preferred servers from invalid BASIC va config", "va_id", out.VAID, "error", err)
		} else {
			out.PreferredServers = basic.PreferredServers
		}
	}
	return out, nil
}

func scanBlock(row interface{ Scan(dest ...any) error }) (*Block, error) {
	var block Block
	var configBytes []byte
	if err := row.Scan(&block.VAID, &block.ConfigType, &configBytes, &block.UpdatedAt, &block.UpdatedBy); err != nil {
		return nil, err
	}
	block.Config = json.RawMessage(configBytes)
	return &block, nil
}
