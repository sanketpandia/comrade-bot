package contact

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, ifCommunityID, message string) (*Submission, error) {
	row := r.db.QueryRowContext(ctx, `
INSERT INTO public.contact_submissions (if_community_id, message)
VALUES ($1, $2)
RETURNING id, if_community_id, message, created_at`,
		ifCommunityID, message,
	)
	var sub Submission
	if err := row.Scan(&sub.ID, &sub.IFCommunityID, &sub.Message, &sub.CreatedAt); err != nil {
		return nil, fmt.Errorf("create contact submission: %w", err)
	}
	return &sub, nil
}
