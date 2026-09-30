package adminvote

import "errors"

var (
	ErrVoteNotOpen      = errors.New("vote is not open")
	ErrVoteExpired      = errors.New("vote has expired")
	ErrAlreadyVoted     = errors.New("admin has already voted on this vote")
	ErrNotYetExpired    = errors.New("vote has not yet passed its expiry")
	ErrNoOpenVote       = errors.New("no open vote for this action and target")
	ErrProposerIsTarget = errors.New("proposer cannot be the target user")
)
