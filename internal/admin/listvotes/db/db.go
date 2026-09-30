package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

const listOpenVotes = `-- name: ListOpenVotes :many
SELECT
    v.id, v.action_type, v.target_user_id, v.status,
    v.required_votes, v.votes_for, v.votes_against,
    v.expires_at, v.created_at,
    u.email        AS target_email,
    u.role         AS target_role,
    COALESCE(d.full_name, p.full_name, '') AS target_full_name
FROM admin_votes v
         JOIN identity_users u ON u.id = v.target_user_id
         LEFT JOIN doctors_profiles  d ON d.user_id = v.target_user_id AND d.deleted_at IS NULL
         LEFT JOIN patients_profiles p ON p.user_id = v.target_user_id AND p.deleted_at IS NULL
WHERE v.status = 'Open' AND v.deleted_at IS NULL
ORDER BY v.expires_at ASC
`

type ListOpenVotesRow struct {
	ID             int64
	ActionType     string
	TargetUserID   int64
	Status         string
	RequiredVotes  int
	VotesFor       int
	VotesAgainst   int
	ExpiresAt      pgtype.Timestamptz
	CreatedAt      pgtype.Timestamptz
	TargetEmail    string
	TargetRole     string
	TargetFullName string
}

func (q *Queries) ListOpenVotes(ctx context.Context) ([]ListOpenVotesRow, error) {
	rows, err := q.db.Query(ctx, listOpenVotes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListOpenVotesRow, 0)
	for rows.Next() {
		var i ListOpenVotesRow
		if err := rows.Scan(
			&i.ID, &i.ActionType, &i.TargetUserID, &i.Status,
			&i.RequiredVotes, &i.VotesFor, &i.VotesAgainst,
			&i.ExpiresAt, &i.CreatedAt,
			&i.TargetEmail, &i.TargetRole, &i.TargetFullName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
