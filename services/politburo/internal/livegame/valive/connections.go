package valive

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	HistoryEventLinked          = "linked"
	HistoryEventUnlinked        = "unlinked"
	HistoryEventSkippedOccupied = "skipped_occupied"
)

// Connection is a persisted link between a VA and an Infinite Flight Live organisation.
type Connection struct {
	ID                 string
	VAID               string
	LiveOrganizationID string
	OrganizationName   string
	OrganizationRole   int
	LinkedByUserID     string
	LinkedAt           time.Time
}

type ConnectionsRepository struct {
	db *sql.DB
}

func NewConnectionsRepository(db *sql.DB) *ConnectionsRepository {
	return &ConnectionsRepository{db: db}
}

const connectionColumns = `id, va_id, live_organization_id, organization_name, organization_role, linked_by_user_id, linked_at`

func (r *ConnectionsRepository) ListByVAID(ctx context.Context, vaID string) ([]Connection, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+connectionColumns+`
FROM public.va_live_connections
WHERE va_id = $1
ORDER BY organization_name`, vaID)
	if err != nil {
		return nil, fmt.Errorf("list va live connections: %w", err)
	}
	defer rows.Close()
	var out []Connection
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *ConnectionsRepository) GetByVAAndOrg(ctx context.Context, vaID, orgID string) (*Connection, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT `+connectionColumns+`
FROM public.va_live_connections
WHERE va_id = $1 AND live_organization_id = $2`, vaID, orgID)
	c, err := scanConnection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *ConnectionsRepository) GetByOrgID(ctx context.Context, orgID string) (*Connection, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT `+connectionColumns+`
FROM public.va_live_connections
WHERE live_organization_id = $1`, orgID)
	c, err := scanConnection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *ConnectionsRepository) Upsert(ctx context.Context, vaID, orgID, name string, role int, linkedBy string) error {
	existing, err := r.GetByOrgID(ctx, orgID)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.VAID != vaID {
			return fmt.Errorf("organization already linked to another VA")
		}
		_, err = r.db.ExecContext(ctx, `
UPDATE public.va_live_connections
SET organization_name = $3, organization_role = $4, linked_by_user_id = $5
WHERE va_id = $1 AND live_organization_id = $2`,
			vaID, orgID, name, role, linkedBy,
		)
		if err != nil {
			return fmt.Errorf("refresh va live connection: %w", err)
		}
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
INSERT INTO public.va_live_connections (va_id, live_organization_id, organization_name, organization_role, linked_by_user_id)
VALUES ($1, $2, $3, $4, $5)`,
		vaID, orgID, name, role, linkedBy,
	)
	if err != nil {
		return fmt.Errorf("insert va live connection: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO public.va_live_connection_history (va_id, live_organization_id, event, actor_user_id, organization_name)
VALUES ($1, $2, $3, $4, $5)`,
		vaID, orgID, HistoryEventLinked, linkedBy, name,
	)
	if err != nil {
		return fmt.Errorf("insert connection history: %w", err)
	}
	return tx.Commit()
}

func (r *ConnectionsRepository) Delete(ctx context.Context, vaID, orgID, actorUserID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var name string
	err = tx.QueryRowContext(ctx, `
DELETE FROM public.va_live_connections
WHERE va_id = $1 AND live_organization_id = $2
RETURNING organization_name`, vaID, orgID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.ErrNoRows
	}
	if err != nil {
		return fmt.Errorf("delete va live connection: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO public.va_live_connection_history (va_id, live_organization_id, event, actor_user_id, organization_name)
VALUES ($1, $2, $3, $4, $5)`,
		vaID, orgID, HistoryEventUnlinked, actorUserID, name,
	)
	if err != nil {
		return fmt.Errorf("insert unlink history: %w", err)
	}
	return tx.Commit()
}

func (r *ConnectionsRepository) RecordSkippedOccupied(ctx context.Context, vaID, orgID, actorUserID, name string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO public.va_live_connection_history (va_id, live_organization_id, event, actor_user_id, organization_name)
VALUES ($1, $2, $3, $4, $5)`,
		vaID, orgID, HistoryEventSkippedOccupied, actorUserID, name,
	)
	if err != nil {
		return fmt.Errorf("insert skipped occupied history: %w", err)
	}
	return nil
}

func scanConnection(row interface{ Scan(dest ...any) error }) (*Connection, error) {
	var c Connection
	if err := row.Scan(
		&c.ID, &c.VAID, &c.LiveOrganizationID, &c.OrganizationName, &c.OrganizationRole,
		&c.LinkedByUserID, &c.LinkedAt,
	); err != nil {
		return nil, fmt.Errorf("scan va live connection: %w", err)
	}
	return &c, nil
}
