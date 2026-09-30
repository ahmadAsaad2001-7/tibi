package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries { return &Queries{db: db} }

type Queries struct{ db DBTX }

const getVoteForUpdate = `-- name: GetVoteForUpdate :one
SELECT
    id, action_type, target_user_id, status, required_votes,
    votes_for, votes_against, expires_at, resolved_at,
    xmin::text AS xmin
FROM admin_votes
WHERE id = $1 AND deleted_at IS NULL
`

type GetVoteForUpdateRow struct {
	ID            int64
	ActionType    string
	TargetUserID  int64
	Status        string
	RequiredVotes int
	VotesFor      int
	VotesAgainst  int
	ExpiresAt     time.Time
	ResolvedAt    *time.Time
	Xmin          string
}

func (q *Queries) GetVoteForUpdate(ctx context.Context, voteID int64) (GetVoteForUpdateRow, error) {
	row := q.db.QueryRow(ctx, getVoteForUpdate, voteID)
	var i GetVoteForUpdateRow
	err := row.Scan(
		&i.ID, &i.ActionType, &i.TargetUserID, &i.Status, &i.RequiredVotes,
		&i.VotesFor, &i.VotesAgainst, &i.ExpiresAt, &i.ResolvedAt, &i.Xmin,
	)
	return i, err
}

const getVoteParticipants = `-- name: GetVoteParticipants :many
SELECT id, admin_vote_id, admin_user_id, vote, voted_at
FROM admin_vote_participants
WHERE admin_vote_id = $1
ORDER BY voted_at
`

type GetVoteParticipantsRow struct {
	ID          int64
	AdminVoteID int64
	AdminUserID int64
	Vote        string
	VotedAt     time.Time
}

func (q *Queries) GetVoteParticipants(ctx context.Context, voteID int64) ([]GetVoteParticipantsRow, error) {
	rows, err := q.db.Query(ctx, getVoteParticipants, voteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]GetVoteParticipantsRow, 0)
	for rows.Next() {
		var i GetVoteParticipantsRow
		if err := rows.Scan(&i.ID, &i.AdminVoteID, &i.AdminUserID, &i.Vote, &i.VotedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateVoteTally = `-- name: UpdateVoteTally :execrows
UPDATE admin_votes
SET votes_for     = $2,
    votes_against = $3,
    status        = $4,
    resolved_at   = $5,
    updated_at    = now()
WHERE id = $1 AND xmin::text = $6
`

type UpdateVoteTallyParams struct {
	ID           int64
	VotesFor     int
	VotesAgainst int
	Status       string
	ResolvedAt   *time.Time
	Xmin         string
}

func (q *Queries) UpdateVoteTally(ctx context.Context, arg UpdateVoteTallyParams) (int64, error) {
	tag, err := q.db.Exec(ctx, updateVoteTally,
		arg.ID, arg.VotesFor, arg.VotesAgainst, arg.Status, arg.ResolvedAt, arg.Xmin)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

const insertVoteParticipant = `-- name: InsertVoteParticipant :exec
INSERT INTO admin_vote_participants (admin_vote_id, admin_user_id, vote, voted_at)
VALUES ($1, $2, $3, $4)
`

type InsertVoteParticipantParams struct {
	AdminVoteID int64
	AdminUserID int64
	Vote        string
	VotedAt     time.Time
}

func (q *Queries) InsertVoteParticipant(ctx context.Context, arg InsertVoteParticipantParams) error {
	_, err := q.db.Exec(ctx, insertVoteParticipant, arg.AdminVoteID, arg.AdminUserID, arg.Vote, arg.VotedAt)
	return err
}
