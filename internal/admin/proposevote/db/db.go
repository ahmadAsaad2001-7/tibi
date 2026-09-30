package db

import (
	"context"
	"time"

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

const findOpenVote = `-- name: FindOpenVote :one
SELECT id FROM admin_votes
WHERE action_type = $1
  AND target_user_id = $2
  AND status = 'Open'
  AND deleted_at IS NULL
`

type FindOpenVoteParams struct {
	ActionType   string
	TargetUserID int64
}

func (q *Queries) FindOpenVote(ctx context.Context, arg FindOpenVoteParams) (int64, error) {
	row := q.db.QueryRow(ctx, findOpenVote, arg.ActionType, arg.TargetUserID)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const insertAdminVote = `-- name: InsertAdminVote :one
INSERT INTO admin_votes (
    action_type, target_user_id, status, required_votes,
    votes_for, votes_against, expires_at, resolved_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at
`

type InsertAdminVoteParams struct {
	ActionType    string
	TargetUserID  int64
	Status        string
	RequiredVotes int
	VotesFor      int
	VotesAgainst  int
	ExpiresAt     time.Time
	ResolvedAt    *time.Time
}

type InsertAdminVoteRow struct {
	ID        int64
	CreatedAt pgtype.Timestamptz
}

func (q *Queries) InsertAdminVote(ctx context.Context, arg InsertAdminVoteParams) (InsertAdminVoteRow, error) {
	row := q.db.QueryRow(ctx, insertAdminVote,
		arg.ActionType, arg.TargetUserID, arg.Status, arg.RequiredVotes,
		arg.VotesFor, arg.VotesAgainst, arg.ExpiresAt, arg.ResolvedAt,
	)
	var i InsertAdminVoteRow
	err := row.Scan(&i.ID, &i.CreatedAt)
	return i, err
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

const findUserRoleForVote = `-- name: FindUserRoleForVote :one
SELECT role FROM identity_users
WHERE id = $1 AND deleted_at IS NULL
`

func (q *Queries) FindUserRoleForVote(ctx context.Context, userID int64) (string, error) {
	row := q.db.QueryRow(ctx, findUserRoleForVote, userID)
	var role string
	err := row.Scan(&role)
	return role, err
}
