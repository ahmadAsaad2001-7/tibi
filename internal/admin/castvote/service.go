package castvote

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/admin/adminvote"
	"tibi/internal/admin/castvote/db"
	doctorscontracts "tibi/internal/doctors/contracts"
	identitycontracts "tibi/internal/identity/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db       *database.DB
	doctors  doctorscontracts.API
	identity identitycontracts.API
	clock    func() time.Time
}

func NewService(db *database.DB, doctors doctorscontracts.API, identity identitycontracts.API) *Service {
	return &Service{db: db, doctors: doctors, identity: identity, clock: time.Now}
}

type Command struct {
	VoteID      int64
	AdminUserID int64
	Choice      adminvote.Choice
}

type Response struct {
	VoteID        int64      `json:"vote_id"`
	Status        string     `json:"status"`
	VotesFor      int        `json:"votes_for"`
	VotesAgainst  int        `json:"votes_against"`
	RequiredVotes int        `json:"required_votes"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	ActionApplied bool       `json:"action_applied"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetVoteForUpdate(ctx, cmd.VoteID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("vote not found")
			}
			return httpx.Internal(err)
		}

		participants, err := q.GetVoteParticipants(ctx, cmd.VoteID)
		if err != nil {
			return httpx.Internal(err)
		}

		vote := hydrateVote(row, participants)

		now := s.clock()
		if err := vote.CastVote(cmd.AdminUserID, cmd.Choice, now); err != nil {
			return mapDomainError(err)
		}

		// Persist the vote row first. If the xmin guard fails, another
		// admin voted between our read and our write. Return 409 without
		// inserting a participant.
		affected, err := q.UpdateVoteTally(ctx, db.UpdateVoteTallyParams{
			ID:           vote.ID,
			VotesFor:     vote.VotesFor,
			VotesAgainst: vote.VotesAgainst,
			Status:       string(vote.Status),
			ResolvedAt:   vote.ResolvedAt,
			Xmin:         row.Xmin,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if affected == 0 {
			return httpx.Conflict("vote was modified by another request; retry")
		}

		// Append the participant. Unique constraint on (vote, admin) is
		// the race guard; the service already checked for duplicates.
		if err := q.InsertVoteParticipant(ctx, db.InsertVoteParticipantParams{
			AdminVoteID: vote.ID,
			AdminUserID: cmd.AdminUserID,
			Vote:        string(cmd.Choice),
			VotedAt:     now,
		}); err != nil {
			return httpx.Internal(err)
		}

		// If the vote just resolved, apply the action. This is a cross-module
		// write and it shares this transaction. If any step fails, the vote
		// rolls back — there is no such thing as a resolved vote whose
		// action did not run.
		applied := false
		if choice := vote.Resolution(); choice != nil && *choice == adminvote.ChoiceFor {
			if err := s.applyAction(ctx, vote); err != nil {
				return err
			}
			applied = true
		}

		resp = &Response{
			VoteID:        vote.ID,
			Status:        string(vote.Status),
			VotesFor:      vote.VotesFor,
			VotesAgainst:  vote.VotesAgainst,
			RequiredVotes: vote.RequiredVotes,
			ResolvedAt:    vote.ResolvedAt,
			ActionApplied: applied,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) applyAction(ctx context.Context, vote *adminvote.Vote) error {
	switch vote.ActionType {
	case adminvote.ActionVerifyDoctor:
		if err := s.doctors.SetVerificationStatus(ctx, doctorscontracts.SetVerificationStatusInput{
			UserID: vote.TargetUserID,
			Status: "Verified",
		}); err != nil {
			return err
		}
		// May be a no-op if the user is already a Doctor, in which case
		// the guard inside setRole returns 409 — which we do not want for
		// an idempotent verify. Handle by attempting demote-on-failure.
		if err := s.identity.PromotePendingDoctorToDoctor(ctx, vote.TargetUserID); err != nil {
			// If the role is already Doctor, this is fine (idempotent verify).
			// If the error is anything else, propagate.
			if !isRoleAlreadyDoctor(err) {
				return err
			}
		}
		return nil

	case adminvote.ActionUnverifyDoctor:
		if err := s.doctors.SetVerificationStatus(ctx, doctorscontracts.SetVerificationStatusInput{
			UserID: vote.TargetUserID,
			Status: "NotSubmitted",
		}); err != nil {
			return err
		}
		if err := s.identity.DemoteDoctorToPending(ctx, vote.TargetUserID); err != nil {
			if !isRoleAlreadyPending(err) {
				return err
			}
		}
		return nil

	case adminvote.ActionBanUser:
		// Slice 4.
		return httpx.Unprocessable("BanUser is not yet supported")
	}
	return httpx.Internal(errors.New("unknown action type"))
}

func hydrateVote(row db.GetVoteForUpdateRow, participants []db.GetVoteParticipantsRow) *adminvote.Vote {
	parts := make([]adminvote.Participant, len(participants))
	for i, p := range participants {
		parts[i] = adminvote.Participant{
			ID:          p.ID,
			AdminVoteID: p.AdminVoteID,
			AdminUserID: p.AdminUserID,
			Vote:        adminvote.Choice(p.Vote),
			VotedAt:     p.VotedAt,
		}
	}
	return &adminvote.Vote{
		ID:            row.ID,
		ActionType:    adminvote.ActionType(row.ActionType),
		TargetUserID:  row.TargetUserID,
		Status:        adminvote.Status(row.Status),
		RequiredVotes: row.RequiredVotes,
		VotesFor:      row.VotesFor,
		VotesAgainst:  row.VotesAgainst,
		ExpiresAt:     row.ExpiresAt,
		ResolvedAt:    row.ResolvedAt,
		Participants:  parts,
	}
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, adminvote.ErrVoteNotOpen):
		return httpx.Conflict("vote is not open")
	case errors.Is(err, adminvote.ErrVoteExpired):
		return httpx.Conflict("vote has expired")
	case errors.Is(err, adminvote.ErrAlreadyVoted):
		return httpx.Conflict("admin has already voted")
	default:
		return httpx.Internal(err)
	}
}

func isRoleAlreadyDoctor(err error) bool {
	return roleConflict(err, "Doctor")
}

func isRoleAlreadyPending(err error) bool {
	return roleConflict(err, "PendingDoctor")
}

func roleConflict(err error, role string) bool {
	he := httpx.As(err)
	return he.Code == "conflict" && strings.Contains(he.Message, "user role is "+role)
}
