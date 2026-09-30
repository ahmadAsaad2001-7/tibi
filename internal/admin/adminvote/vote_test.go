package adminvote

import (
	"testing"
	"time"
)

func TestNewVoteCountsProposerAsFor(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionVerifyDoctor, 10, 20, 2, now.Add(time.Hour), now)

	if vote.VotesFor != 1 || vote.VotesAgainst != 0 {
		t.Fatalf("tally = %d/%d, want 1/0", vote.VotesFor, vote.VotesAgainst)
	}
	if vote.Status != StatusOpen {
		t.Fatalf("status = %s, want Open", vote.Status)
	}
	if vote.Resolution() != nil {
		t.Fatal("open vote should not have a resolution")
	}
}

func TestSecondForVoteResolves(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionVerifyDoctor, 10, 20, 2, now.Add(time.Hour), now)
	if err := vote.CastVote(21, ChoiceFor, now); err != nil {
		t.Fatal(err)
	}
	if vote.Status != StatusResolved {
		t.Fatalf("status = %s, want Resolved", vote.Status)
	}
	choice := vote.Resolution()
	if choice == nil || *choice != ChoiceFor {
		t.Fatalf("resolution = %v, want For", choice)
	}
}

func TestDuplicateVoteRejected(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionVerifyDoctor, 10, 20, 2, now.Add(time.Hour), now)
	err := vote.CastVote(20, ChoiceAgainst, now)
	if err != ErrAlreadyVoted {
		t.Fatalf("err = %v, want ErrAlreadyVoted", err)
	}
}

func TestExpiredVoteRejected(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionVerifyDoctor, 10, 20, 2, now.Add(-time.Minute), now.Add(-time.Hour))
	err := vote.CastVote(21, ChoiceFor, now)
	if err != ErrVoteExpired {
		t.Fatalf("err = %v, want ErrVoteExpired", err)
	}
}

func TestAgainstMajorityResolves(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionUnverifyDoctor, 10, 20, 2, now.Add(time.Hour), now)
	if err := vote.CastVote(21, ChoiceAgainst, now); err != nil {
		t.Fatal(err)
	}
	if vote.Status != StatusOpen {
		t.Fatal("one against vote must leave a two-vote motion open")
	}
	if err := vote.CastVote(22, ChoiceAgainst, now); err != nil {
		t.Fatal(err)
	}
	if vote.Status != StatusResolved || vote.VotesFor != 1 || vote.VotesAgainst != 2 {
		t.Fatalf("status=%s for=%d against=%d", vote.Status, vote.VotesFor, vote.VotesAgainst)
	}
	choice := vote.Resolution()
	if choice == nil || *choice != ChoiceAgainst {
		t.Fatalf("resolution = %v, want Against", choice)
	}
}

func TestSingleRequiredVoteResolvesOnProposal(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionVerifyDoctor, 10, 20, 1, now.Add(time.Hour), now)
	if vote.Status != StatusResolved {
		t.Fatalf("status = %s, want Resolved", vote.Status)
	}
	choice := vote.Resolution()
	if choice == nil || *choice != ChoiceFor {
		t.Fatalf("resolution = %v, want For", choice)
	}
}

func TestCastOnClosedVote(t *testing.T) {
	now := time.Now()
	vote := NewVote(ActionVerifyDoctor, 10, 20, 1, now.Add(time.Hour), now)
	err := vote.CastVote(21, ChoiceFor, now)
	if err != ErrVoteNotOpen {
		t.Fatalf("err = %v, want ErrVoteNotOpen", err)
	}
}

func TestExpire(t *testing.T) {
	created := time.Now().Add(-2 * time.Hour)
	vote := NewVote(ActionVerifyDoctor, 10, 20, 2, created.Add(time.Hour), created)
	if err := vote.Expire(created.Add(30 * time.Minute)); err != ErrNotYetExpired {
		t.Fatalf("early expire err = %v", err)
	}
	now := created.Add(2 * time.Hour)
	if err := vote.Expire(now); err != nil {
		t.Fatal(err)
	}
	if vote.Status != StatusExpired || vote.ResolvedAt == nil || !vote.ResolvedAt.Equal(now) {
		t.Fatalf("expired vote = %+v", vote)
	}
	if err := vote.Expire(now); err != ErrVoteNotOpen {
		t.Fatalf("second expire err = %v", err)
	}
}

func TestActionTypeValid(t *testing.T) {
	for _, action := range []ActionType{ActionVerifyDoctor, ActionUnverifyDoctor, ActionBanUser} {
		if !action.Valid() {
			t.Fatalf("%s should be valid", action)
		}
	}
	if ActionType("DeleteEverything").Valid() {
		t.Fatal("unknown action reported valid")
	}
}
