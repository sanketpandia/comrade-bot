package pirepvaluemappings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const rowColumns = `va_id, aircraft_mappings, airline_mappings, default_aircraft, default_airline, revision, updated_at, updated_by`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type Row struct {
	VAID             string
	AircraftMappings map[string]string
	AirlineMappings  map[string]string
	DefaultAircraft  *string
	DefaultAirline   *string
	Revision         int64
	UpdatedAt        time.Time
	UpdatedBy        string
}

func (r *Repository) Get(ctx context.Context, vaID string) (*Row, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+rowColumns+` FROM public.va_pirep_value_mappings WHERE va_id = $1`, vaID)
	out, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get pirep value mappings: %w", err)
	}
	return out, nil
}

func (r *Repository) UpsertIfRevision(ctx context.Context, vaID string, payload Row, expectedRevision int64, updatedBy string) (*Row, error) {
	aircraftJSON, err := json.Marshal(payload.AircraftMappings)
	if err != nil {
		return nil, err
	}
	airlineJSON, err := json.Marshal(payload.AirlineMappings)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	if expectedRevision == 0 {
		newRev := int64(1)
		row := r.db.QueryRowContext(ctx, `
INSERT INTO public.va_pirep_value_mappings (
	va_id, aircraft_mappings, airline_mappings, default_aircraft, default_airline, revision, updated_at, updated_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (va_id) DO NOTHING
RETURNING `+rowColumns,
			vaID, aircraftJSON, airlineJSON, payload.DefaultAircraft, payload.DefaultAirline, newRev, now, updatedBy,
		)
		out, err := scanRow(row)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConflict
		}
		if err != nil {
			return nil, fmt.Errorf("insert pirep value mappings: %w", err)
		}
		return out, nil
	}

	newRev := expectedRevision + 1
	result, err := r.db.ExecContext(ctx, `
UPDATE public.va_pirep_value_mappings
SET aircraft_mappings = $3,
    airline_mappings = $4,
    default_aircraft = $5,
    default_airline = $6,
    revision = $7,
    updated_at = $8,
    updated_by = $9
WHERE va_id = $1 AND revision = $2`,
		vaID, expectedRevision, aircraftJSON, airlineJSON, payload.DefaultAircraft, payload.DefaultAirline, newRev, now, updatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("update pirep value mappings: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrConflict
	}
	return r.Get(ctx, vaID)
}

func scanRow(row *sql.Row) (*Row, error) {
	var out Row
	var aircraftRaw, airlineRaw []byte
	if err := row.Scan(
		&out.VAID, &aircraftRaw, &airlineRaw, &out.DefaultAircraft, &out.DefaultAirline,
		&out.Revision, &out.UpdatedAt, &out.UpdatedBy,
	); err != nil {
		return nil, err
	}
	out.AircraftMappings = map[string]string{}
	out.AirlineMappings = map[string]string{}
	if len(aircraftRaw) > 0 {
		if err := json.Unmarshal(aircraftRaw, &out.AircraftMappings); err != nil {
			return nil, fmt.Errorf("decode aircraft mappings: %w", err)
		}
	}
	if len(airlineRaw) > 0 {
		if err := json.Unmarshal(airlineRaw, &out.AirlineMappings); err != nil {
			return nil, fmt.Errorf("decode airline mappings: %w", err)
		}
	}
	return &out, nil
}