package liveries

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

const liveryColumns = `livery_id, aircraft_id, aircraft_name, livery_name, display_aircraft_name, display_livery_name, updated_at`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertFromUpstream(ctx context.Context, upstream []infiniteflight.Livery) (int, error) {
	if len(upstream) == 0 {
		return 0, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin livery upsert tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO public.game_aircraft_liveries (
			livery_id, aircraft_id, aircraft_name, livery_name,
			display_aircraft_name, display_livery_name, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $3, $4, $5, $5)
		ON CONFLICT (livery_id) DO UPDATE SET
			aircraft_id = EXCLUDED.aircraft_id,
			aircraft_name = EXCLUDED.aircraft_name,
			livery_name = EXCLUDED.livery_name,
			updated_at = EXCLUDED.updated_at`)
	if err != nil {
		return 0, fmt.Errorf("prepare livery upsert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for _, item := range upstream {
		if item.ID == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, item.ID, item.AircraftID, item.AircraftName, item.LiveryName, now); err != nil {
			return 0, fmt.Errorf("upsert livery %s: %w", item.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit livery upsert: %w", err)
	}
	return len(upstream), nil
}

func (r *Repository) ListAll(ctx context.Context) ([]AircraftLivery, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+liveryColumns+` FROM public.game_aircraft_liveries`)
	if err != nil {
		return nil, fmt.Errorf("list liveries: %w", err)
	}
	defer rows.Close()

	result := make([]AircraftLivery, 0)
	for rows.Next() {
		var row AircraftLivery
		if err := rows.Scan(
			&row.LiveryID, &row.AircraftID, &row.AircraftName, &row.LiveryName,
			&row.DisplayAircraftName, &row.DisplayLiveryName, &row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan livery: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate liveries: %w", err)
	}
	return result, nil
}
