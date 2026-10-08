package proposesuspension

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tibi/internal/admin/adminvote"
	"tibi/internal/admin/proposesuspension/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Response struct {
	VoteID        int64  `json:"vote_id"`
	Status        string `json:"status"`
	VotesFor      int    `json:"votes_for"`
	VotesAgainst  int    `json:"votes_against"`
	RequiredVotes int    `json:"required_votes"`
	Resolved      bool   `json:"resolved"`
}

func (s *Service) Execute(ctx context.Context, proposerAdminID, targetUserID int64, cmd Command) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		if _, err := q.GetSuspensionTarget(ctx, targetUserID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("user not found")
			}
			return httpx.Internal(err)
		}

		has, err := q.HasActiveSuspension(ctx, targetUserID)
		if err != nil {
			return httpx.Internal(err)
		}
		if has {
			return httpx.AlreadyExists("user is already suspended")
		}

		payload, err := json.Marshal(map[string]any{
			"reason": cmd.Reason,
			"days":   cmd.Days,
		})
		if err != nil {
			return httpx.Internal(err)
		}

		now := s.clock()
		expiresAt := now.Add(adminvote.DefaultTTL)

		row, err := q.InsertSuspensionVote(ctx, db.InsertSuspensionVoteParams{
			TargetUserID:  targetUserID,
			RequiredVotes: int32(adminvote.RequiredVotes),
			ExpiresAt:     pgTime(expiresAt),
			Payload:       payload,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return httpx.AlreadyExists("a suspension vote is already open for this user")
			}
			return httpx.Internal(err)
		}

		if err := q.InsertSuspensionVoteParticipant(ctx, db.InsertSuspensionVoteParticipantParams{
			AdminVoteID: row.ID,
			AdminUserID: proposerAdminID,
			VotedAt:     pgTime(now),
		}); err != nil {
			return httpx.Internal(err)
		}

		resp = &Response{
			VoteID:        row.ID,
			Status:        "Open",
			VotesFor:      1,
			VotesAgainst:  0,
			RequiredVotes: adminvote.RequiredVotes,
			Resolved:      false,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}