package membership

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetForUserAndVA(ctx context.Context, userID, vaID string) (*Membership, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT vur.id, vur.user_id, vur.va_id, vur.role::text, vur.is_active, vur.callsign, vur.airtable_pilot_id, vur.joined_at,
       va.name, va.code
FROM public.va_user_roles vur
JOIN public.virtual_airlines va ON va.id = vur.va_id
WHERE vur.user_id = $1 AND vur.va_id = $2`,
		userID, vaID,
	)
	m, err := scanMembership(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *Repository) GetActiveForUserAndVA(ctx context.Context, userID, vaID string) (*Membership, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT vur.id, vur.user_id, vur.va_id, vur.role::text, vur.is_active, vur.callsign, vur.airtable_pilot_id, vur.joined_at,
       va.name, va.code
FROM public.va_user_roles vur
JOIN public.virtual_airlines va ON va.id = vur.va_id
WHERE vur.user_id = $1 AND vur.va_id = $2 AND vur.is_active = true`,
		userID, vaID,
	)
	m, err := scanMembership(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]Membership, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT vur.id, vur.user_id, vur.va_id, vur.role::text, vur.is_active, vur.callsign, vur.airtable_pilot_id, vur.joined_at,
       va.name, va.code
FROM public.va_user_roles vur
JOIN public.virtual_airlines va ON va.id = vur.va_id
WHERE vur.user_id = $1
ORDER BY vur.joined_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()
	return scanMembershipRows(rows)
}

type JoinInput struct {
	UserID          string
	VAID            string
	Role            string
	Callsign        string
	AirtablePilotID string
}

func (r *Repository) Join(ctx context.Context, input JoinInput) (*Membership, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.va_user_roles (user_id, va_id, role, is_active, callsign, airtable_pilot_id)
VALUES ($1, $2, $3::public.va_role, true, $4, nullif($5, ''))
RETURNING id, user_id, va_id, role::text, is_active, callsign, airtable_pilot_id, joined_at`,
		input.UserID, input.VAID, input.Role, input.Callsign, input.AirtablePilotID,
	)
	base, err := scanMembershipBase(row)
	if err != nil {
		return nil, err
	}
	va, err := r.loadVA(ctx, input.VAID)
	if err != nil {
		return nil, err
	}
	base.VAName = va.Name
	base.VACode = va.Code
	return base, nil
}

func (r *Repository) loadVA(ctx context.Context, vaID string) (struct{ Name, Code string }, error) {
	var name, code string
	err := r.db.QueryRowContext(ctx, `SELECT name, code FROM public.virtual_airlines WHERE id = $1`, vaID).Scan(&name, &code)
	if err != nil {
		return struct{ Name, Code string }{}, fmt.Errorf("load va: %w", err)
	}
	return struct{ Name, Code string }{Name: name, Code: code}, nil
}

func scanMembershipRows(rows *sql.Rows) ([]Membership, error) {
	var out []Membership
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func scanMembership(row interface{ Scan(dest ...any) error }) (*Membership, error) {
	var m Membership
	var callsign, airtablePilotID sql.NullString
	err := row.Scan(
		&m.ID, &m.UserID, &m.VAID, &m.Role, &m.IsActive, &callsign, &airtablePilotID, &m.JoinedAt,
		&m.VAName, &m.VACode,
	)
	if err != nil {
		return nil, fmt.Errorf("scan membership: %w", err)
	}
	m.Callsign = nullString(callsign)
	m.AirtablePilotID = nullString(airtablePilotID)
	return &m, nil
}

func scanMembershipBase(row interface{ Scan(dest ...any) error }) (*Membership, error) {
	var m Membership
	var callsign, airtablePilotID sql.NullString
	err := row.Scan(
		&m.ID, &m.UserID, &m.VAID, &m.Role, &m.IsActive, &callsign, &airtablePilotID, &m.JoinedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan membership: %w", err)
	}
	m.Callsign = nullString(callsign)
	m.AirtablePilotID = nullString(airtablePilotID)
	return &m, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	s := value.String
	return &s
}

func (r *Repository) ListForVA(ctx context.Context, vaID string, active bool) ([]Member, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.if_community_id, u.username,
       vur.id, vur.role::text, vur.is_active, vur.callsign, vur.airtable_pilot_id,
       vur.joined_at, vur.updated_at
FROM public.va_user_roles vur
JOIN public.users u ON u.id = vur.user_id
WHERE vur.va_id = $1 AND vur.is_active = $2
ORDER BY vur.joined_at DESC`,
		vaID, active,
	)
	if err != nil {
		return nil, fmt.Errorf("list va members: %w", err)
	}
	defer rows.Close()

	var out []Member
	for rows.Next() {
		var m Member
		var ifCommunityID, username, callsign, airtablePilotID sql.NullString
		if err := rows.Scan(
			&m.UserID, &ifCommunityID, &username,
			&m.MembershipID, &m.Role, &m.IsActive, &callsign, &airtablePilotID,
			&m.JoinedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan va member: %w", err)
		}
		m.IFCommunityID = nullString(ifCommunityID)
		m.Username = nullString(username)
		m.Callsign = nullString(callsign)
		m.AirtablePilotID = nullString(airtablePilotID)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateRoleForVA(ctx context.Context, vaID, targetUserID, role string, demoteAdminUserID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if demoteAdminUserID != "" {
		res, err := tx.ExecContext(ctx, `
UPDATE public.va_user_roles
SET role = $3::public.va_role, updated_at = (now() AT TIME ZONE 'UTC')
WHERE user_id = $1 AND va_id = $2 AND is_active = true AND role = 'administrator'::public.va_role`,
			demoteAdminUserID, vaID, RoleBourgeoisie,
		)
		if err != nil {
			return fmt.Errorf("demote administrator: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return sql.ErrNoRows
		}
	}

	res, err := tx.ExecContext(ctx, `
UPDATE public.va_user_roles
SET role = $3::public.va_role, updated_at = (now() AT TIME ZONE 'UTC')
WHERE user_id = $1 AND va_id = $2 AND is_active = true`,
		targetUserID, vaID, role,
	)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *Repository) DeactivateForVA(ctx context.Context, vaID, userID string) error {
	res, err := r.db.ExecContext(ctx, `
UPDATE public.va_user_roles
SET is_active = false, updated_at = (now() AT TIME ZONE 'UTC')
WHERE user_id = $1 AND va_id = $2 AND is_active = true`,
		userID, vaID,
	)
	if err != nil {
		return fmt.Errorf("deactivate membership: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ReactivateForVA(ctx context.Context, vaID, userID string, input JoinInput) (*Membership, error) {
	row := r.db.QueryRowContext(ctx, `
UPDATE public.va_user_roles
SET role = $3::public.va_role, is_active = true,
    callsign = $4, airtable_pilot_id = nullif($5, ''),
    updated_at = (now() AT TIME ZONE 'UTC')
WHERE user_id = $1 AND va_id = $2 AND is_active = false
RETURNING id, user_id, va_id, role::text, is_active, callsign, airtable_pilot_id, joined_at`,
		userID, vaID, input.Role, input.Callsign, input.AirtablePilotID,
	)
	base, err := scanMembershipBase(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	va, err := r.loadVA(ctx, vaID)
	if err != nil {
		return nil, err
	}
	base.VAName = va.Name
	base.VACode = va.Code
	return base, nil
}

func (r *Repository) ActiveAdministratorUserID(ctx context.Context, vaID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, `
SELECT user_id FROM public.va_user_roles
WHERE va_id = $1 AND is_active = true AND role = 'administrator'::public.va_role`,
		vaID,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("lookup administrator: %w", err)
	}
	return userID, nil
}
