package liveries

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
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

// ListForCatalog returns distinct catalog rows for editor browse (paginated).
// kind is "aircraft" or "airline".
func (r *Repository) ListForCatalog(ctx context.Context, kind, search string, limit, offset int) ([]AircraftLivery, int, error) {
	nameCol := "aircraft_name"
	order := "aircraft_name, livery_name, livery_id"
	if kind == "airline" {
		nameCol = "livery_name"
		order = "livery_name, aircraft_name, livery_id"
	}
	where := ""
	args := []any{}
	argN := 1
	if strings.TrimSpace(search) != "" {
		where = fmt.Sprintf(" WHERE %s ILIKE $%d", nameCol, argN)
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		argN++
	}
	countQuery := `SELECT COUNT(DISTINCT ` + nameCol + `) FROM public.game_aircraft_liveries` + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count catalog: %w", err)
	}
	cte := fmt.Sprintf(`
WITH page AS (
	SELECT DISTINCT %s AS sort_name
	FROM public.game_aircraft_liveries%s
	ORDER BY sort_name
	LIMIT $%d OFFSET $%d
)
SELECT DISTINCT ON (g.%s) `+liveryColumns+`
FROM public.game_aircraft_liveries g
JOIN page p ON g.%s = p.sort_name
ORDER BY g.%s, %s`, nameCol, where, argN, argN+1, nameCol, nameCol, nameCol, order)
	query := cte
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list catalog: %w", err)
	}
	defer rows.Close()
	result := make([]AircraftLivery, 0)
	for rows.Next() {
		var row AircraftLivery
		if err := rows.Scan(
			&row.LiveryID, &row.AircraftID, &row.AircraftName, &row.LiveryName,
			&row.DisplayAircraftName, &row.DisplayLiveryName, &row.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan catalog row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (r *Repository) WriteCatalogCSV(ctx context.Context, w io.Writer) error {
	rows, err := r.db.QueryContext(ctx, `
SELECT livery_id, aircraft_id, aircraft_name, livery_name, display_aircraft_name, display_livery_name
FROM public.game_aircraft_liveries
ORDER BY aircraft_name, livery_name, livery_id`)
	if err != nil {
		return fmt.Errorf("query catalog csv: %w", err)
	}
	defer rows.Close()
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"LiveryId", "AircraftId", "AircraftName", "LiveryName", "DisplayAircraftName", "DisplayLiveryName"}); err != nil {
		return err
	}
	for rows.Next() {
		var id, aid, an, ln, dan, dln string
		if err := rows.Scan(&id, &aid, &an, &ln, &dan, &dln); err != nil {
			return err
		}
		if err := cw.Write([]string{id, aid, an, ln, dan, dln}); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}
