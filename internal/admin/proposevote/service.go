package proposevote

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/admin/adminvote"
	"tibi/internal/admin/proposevote/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

const (
	requiredVotes = 2
	voteWindow    = 72 * time.Hour
)

type Service struct {
	db    *database.DB
	clock func() time.Time
	// For the future: when a vote resolves immediately (required_votes=1),
	// the action applies here. Today required_votes=2 so no dispatch happens.
	// (Kept as a seam, not a placeholder.)
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Input struct {
	Action       adminvote.ActionType
	AdminUserID  int64
	TargetUserID int64
}

type Response struct {
	VoteID        int64     `json:"vote_id"`
	ActionType    string    `json:"action_type"`
	TargetUserID  int64     `json:"target_user_id"`
	Status        string    `json:"status"`
	VotesFor      int       `json:"votes_for"`
	VotesAgainst  int       `json:"votes_against"`
	RequiredVotes int       `json:"required_votes"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func (s *Service) Execute(ctx context.Context, in Input) (*Response, error) {
	if !in.Action.Valid() {
		return nil, httpx.ValidationFailed(map[string]string{
			"action_type": "unknown action type",
		})
	}
	if in.AdminUserID == in.TargetUserID {
		return nil, httpx.ValidationFailed(map[string]string{
			"target_user_id": "cannot propose an action on yourself",
		})
	}

	var resp *Response
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		// Precondition: target exists and is in a state that makes sense.
		targetRole, err := q.FindUserRoleForVote(ctx, in.TargetUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("target user not found")
			}
			return httpx.Internal(err)
		}
		if err := s.checkTargetEligibility(in.Action, string(targetRole)); err != nil {
			return err
		}

		// Duplicate open vote check — the DB index is the race guard,
		// this is the friendly error.
		if _, err := q.FindOpenVote(ctx, db.FindOpenVoteParams{
			ActionType:   string(in.Action),
			TargetUserID: in.TargetUserID,
		}); err == nil {
			return httpx.AlreadyExists("an open vote already exists for this action and target")
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return httpx.Internal(err)
		}

		now := s.clock()
		vote := adminvote.NewVote(
			in.Action, in.TargetUserID, in.AdminUserID,
			requiredVotes, now.Add(voteWindow), now,
		)

		row, err := q.InsertAdminVote(ctx, db.InsertAdminVoteParams{
			ActionType:    string(vote.ActionType),
			TargetUserID:  vote.TargetUserID,
			Status:        string(vote.Status),
			RequiredVotes: vote.RequiredVotes,
			VotesFor:      vote.VotesFor,
			VotesAgainst:  vote.VotesAgainst,
			ExpiresAt:     vote.ExpiresAt,
			ResolvedAt:    vote.ResolvedAt,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		vote.ID = row.ID

		if err := q.InsertVoteParticipant(ctx, db.InsertVoteParticipantParams{
			AdminVoteID: vote.ID,
			AdminUserID: vote.Participants[0].AdminUserID,
			Vote:        string(vote.Participants[0].Vote),
			VotedAt:     vote.Participants[0].VotedAt,
		}); err != nil {
			return httpx.Internal(err)
		}

		resp = &Response{
			VoteID:        vote.ID,
			ActionType:    string(vote.ActionType),
			TargetUserID:  vote.TargetUserID,
			Status:        string(vote.Status),
			VotesFor:      vote.VotesFor,
			VotesAgainst:  vote.VotesAgainst,
			RequiredVotes: vote.RequiredVotes,
			ExpiresAt:     vote.ExpiresAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// checkTargetEligibility enforces, per action, what the target's current
// state must be. This is domain knowledge, but it needs to read
// identity_users.role, which is why it happens in the slice rather than
// inside the aggregate.
func (s *Service) checkTargetEligibility(action adminvote.ActionType, currentRole string) error {
	switch action {
	case adminvote.ActionVerifyDoctor:
		if currentRole != "PendingDoctor" && currentRole != "Doctor" {
			return httpx.Unprocessable("target is not a doctor")
		}
	case adminvote.ActionUnverifyDoctor:
		if currentRole != "Doctor" {
			return httpx.Unprocessable("target is not a verified doctor")
		}
	case adminvote.ActionBanUser:
		// Supported by the schema, not yet by slice code.
		return httpx.Unprocessable("BanUser votes are not yet supported")
	}
	return nil
}
