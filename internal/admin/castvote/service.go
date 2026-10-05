package castvote

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

		// ✅ هنا سيعيد []db.AdminVoteParticipant مباشرة
		participants, err := q.GetVoteParticipants(ctx, cmd.VoteID)
		if err != nil {
			return httpx.Internal(err)
		}

		vote := hydrateVote(row, participants)

		now := s.clock()
		if err := vote.CastVote(cmd.AdminUserID, cmd.Choice, now); err != nil {
			return mapDomainError(err)
		}

		resolvedAtTz := pgtype.Timestamptz{Valid: false}
		if vote.ResolvedAt != nil {
			resolvedAtTz = pgtype.Timestamptz{Time: *vote.ResolvedAt, Valid: true}
		}

		var xmin pgtype.Uint32
		if row.Xmin != "" {
			val, err := strconv.ParseUint(row.Xmin, 10, 32)
			if err == nil {
				xmin = pgtype.Uint32{Uint32: uint32(val), Valid: true}
			}
		}

		affected, err := q.UpdateVoteTally(ctx, db.UpdateVoteTallyParams{
			ID:           vote.ID,
			VotesFor:     int32(vote.VotesFor),
			VotesAgainst: int32(vote.VotesAgainst),
			Status:       db.VoteStatus(vote.Status),
			ResolvedAt:   resolvedAtTz,
			Xmin:         xmin, // ✅ مباشر كـ string كما يتوقعه الاستعلام
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if affected == 0 {
			return httpx.Conflict("vote was modified by another request; retry")
		}

		if err := q.InsertVoteParticipant(ctx, db.InsertVoteParticipantParams{
			AdminVoteID: vote.ID,
			AdminUserID: cmd.AdminUserID,
			Vote:        db.VoteChoice(cmd.Choice),
			VotedAt:     pgtype.Timestamptz{Time: now, Valid: true},
		}); err != nil {
			return httpx.Internal(err)
		}

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
		if err := s.identity.PromotePendingDoctorToDoctor(ctx, vote.TargetUserID); err != nil {
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
		return httpx.Unprocessable("BanUser is not yet supported")
	}
	return httpx.Internal(errors.New("unknown action type"))
}

// ✅ FIX الجذري: تغيير النوع من GetVoteParticipantsRow إلى AdminVoteParticipant
func hydrateVote(row db.GetVoteForUpdateRow, participants []db.AdminVoteParticipant) *adminvote.Vote {
	parts := make([]adminvote.Participant, len(participants))
	for i, p := range participants {
		parts[i] = adminvote.Participant{
			ID:          p.ID,
			AdminVoteID: p.AdminVoteID,
			AdminUserID: p.AdminUserID,
			Vote:        adminvote.Choice(p.Vote),
			VotedAt:     p.VotedAt.Time,
		}
	}

	var resolvedAt *time.Time
	if row.ResolvedAt.Valid {
		t := row.ResolvedAt.Time
		resolvedAt = &t
	}

	return &adminvote.Vote{
		ID:            row.ID,
		ActionType:    adminvote.ActionType(row.ActionType),
		TargetUserID:  row.TargetUserID,
		Status:        adminvote.Status(row.Status),
		RequiredVotes: int(row.RequiredVotes),
		VotesFor:      int(row.VotesFor),
		VotesAgainst:  int(row.VotesAgainst),
		ExpiresAt:     row.ExpiresAt.Time,
		ResolvedAt:    resolvedAt,
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
