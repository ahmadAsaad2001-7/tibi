package proposevote

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

		// ✅ FIX 1: استخدام FindOpenVoteParams
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

		resolvedAtTz := pgtype.Timestamptz{Valid: false}
		if vote.ResolvedAt != nil {
			resolvedAtTz = pgtype.Timestamptz{Time: *vote.ResolvedAt, Valid: true}
		}
		// ✅ FIX 2: تحويل الأنواع إلى ما يتوقعه sqlc (db.VoteStatus, int32, pgtype.Timestamptz)
		row, err := q.InsertAdminVote(ctx, db.InsertAdminVoteParams{
			ActionType:    string(vote.ActionType),
			TargetUserID:  vote.TargetUserID,
			Status:        db.VoteStatus(vote.Status),
			RequiredVotes: int32(vote.RequiredVotes),
			VotesFor:      int32(vote.VotesFor),
			VotesAgainst:  int32(vote.VotesAgainst),
			ExpiresAt:     pgtype.Timestamptz{Time: vote.ExpiresAt, Valid: true},
			ResolvedAt:    resolvedAtTz, // ✅ الآن النوع متطابق تماماً
		})
		if err != nil {
			return httpx.Internal(err)
		}
		vote.ID = row.ID

		// ✅ FIX 3: استخدام InsertVoteParticipantParams وتحويل Vote إلى db.VoteChoice
		if err := q.InsertVoteParticipant(ctx, db.InsertVoteParticipantParams{
			AdminVoteID: vote.ID,
			AdminUserID: vote.Participants[0].AdminUserID,
			Vote:        db.VoteChoice(vote.Participants[0].Vote),
			VotedAt:     pgtype.Timestamptz{Time: vote.Participants[0].VotedAt, Valid: true},
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
		return httpx.Unprocessable("BanUser votes are not yet supported")
	}
	return nil
}
