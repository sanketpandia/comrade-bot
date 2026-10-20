package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

func scanBlock(row interface{ Scan(dest ...any) error }) (*Block, error) {
	var block Block
	var configBytes []byte
	if err := row.Scan(&block.VAID, &block.ConfigType, &configBytes, &block.UpdatedAt, &block.UpdatedBy); err != nil {
		return nil, err
	}
	block.Config = json.RawMessage(configBytes)
	return &block, nil
}
