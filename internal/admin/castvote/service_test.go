package castvote

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/admin/adminvote"
	"tibi/internal/admin/castvote/db"
	"tibi/internal/platform/httpx"
)

func TestMapDomainError(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{adminvote.ErrVoteNotOpen, "conflict"},
		{adminvote.ErrVoteExpired, "conflict"},
		{adminvote.ErrAlreadyVoted, "conflict"},
		{errors.New("db down"), "internal_error"},
	}
	for _, tc := range cases {
		he := httpx.As(mapDomainError(tc.err))
		if he.Code != tc.code {
			t.Fatalf("%v -> %s, want %s", tc.err, he.Code, tc.code)
		}
	}
}

func TestRoleConflictDoesNotConfusePendingWithDoctor(t *testing.T) {
	doctor := httpx.Conflict("user role is Doctor, expected PendingDoctor")
	pending := httpx.Conflict("user role is PendingDoctor, expected Doctor")
	if !isRoleAlreadyDoctor(doctor) || isRoleAlreadyPending(doctor) {
		t.Fatal("Doctor conflict was classified incorrectly")
	}
	if !isRoleAlreadyPending(pending) || isRoleAlreadyDoctor(pending) {
		t.Fatal("PendingDoctor conflict was classified incorrectly")
	}
	if isRoleAlreadyDoctor(errors.New("user role is Doctor")) {
		t.Fatal("plain error should not count as a role conflict")
	}
}

func TestApplyActionErrors(t *testing.T) {
	svc := &Service{}
	// ActionBanUser is implemented: a bare vote (no reason payload, no
	// configured suspension service) is a server-side data problem.
	err := svc.applyAction(t.Context(), &adminvote.Vote{ActionType: adminvote.ActionBanUser})
	if code := httpx.As(err).Code; code != "internal_error" {
		t.Fatalf("ban err = %v (code %s, want internal_error)", err, code)
	}
	err = svc.applyAction(t.Context(), &adminvote.Vote{ActionType: "Unknown"})
	if httpx.As(err).Code != "internal_error" {
		t.Fatalf("unknown err = %v", err)
	}
}

func TestHydrateVote(t *testing.T) {
	resolved := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	voted := resolved.Add(-time.Minute)
	vote := hydrateVote(db.GetVoteForUpdateRow{
		ID: 7, ActionType: "VerifyDoctor", TargetUserID: 3, Status: "Resolved",
		RequiredVotes: 2, VotesFor: 2,
		ExpiresAt:  pgtype.Timestamptz{Time: resolved.Add(time.Hour), Valid: true},
		ResolvedAt: pgtype.Timestamptz{Time: resolved, Valid: true},
		Xmin:       "99",
	}, []db.AdminVoteParticipant{{
		ID: 1, AdminVoteID: 7, AdminUserID: 4, Vote: db.VoteChoiceFor, VotedAt: pgtype.Timestamptz{Time: voted, Valid: true},
	}})
	if vote.ID != 7 || vote.Status != adminvote.StatusResolved || len(vote.Participants) != 1 {
		t.Fatalf("vote = %+v", vote)
	}
	if vote.Participants[0].Vote != adminvote.ChoiceFor || vote.Participants[0].AdminUserID != 4 {
		t.Fatalf("participant = %+v", vote.Participants[0])
	}
}
