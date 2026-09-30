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
