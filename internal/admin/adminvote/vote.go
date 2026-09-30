package adminvote

import "time"

type ActionType string

const (
	ActionVerifyDoctor   ActionType = "VerifyDoctor"
	ActionUnverifyDoctor ActionType = "UnverifyDoctor"
	ActionBanUser        ActionType = "BanUser"
)

func (a ActionType) Valid() bool {
	switch a {
	case ActionVerifyDoctor, ActionUnverifyDoctor, ActionBanUser:
		return true
	}
	return false
}

type Status string

const (
	StatusOpen     Status = "Open"
	StatusResolved Status = "Resolved"
	StatusExpired  Status = "Expired"
)

type Choice string

const (
	ChoiceFor     Choice = "For"
	ChoiceAgainst Choice = "Against"
)

type Vote struct {
	ID            int64
	ActionType    ActionType
	TargetUserID  int64
	Status        Status
	RequiredVotes int
	VotesFor      int
	VotesAgainst  int
	ExpiresAt     time.Time
	ResolvedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time

	// Loaded eagerly. Append-only; only CastVote adds to it.
	Participants []Participant
}

type Participant struct {
	ID          int64
	AdminVoteID int64
	AdminUserID int64
	Vote        Choice
	VotedAt     time.Time
}

// NewVote creates a vote with the proposer already counted as a For vote.
// Enforces D-015: a proposal is a vote.
func NewVote(
	action ActionType,
	targetUserID int64,
	proposerUserID int64,
	requiredVotes int,
	expiresAt time.Time,
	now time.Time,
) *Vote {
	v := &Vote{
		ActionType:    action,
		TargetUserID:  targetUserID,
		Status:        StatusOpen,
		RequiredVotes: requiredVotes,
		ExpiresAt:     expiresAt,
	}
	v.Participants = []Participant{{
		AdminUserID: proposerUserID,
		Vote:        ChoiceFor,
		VotedAt:     now,
	}}
	v.VotesFor = 1
	v.maybeResolve(now)
	return v
}

// CastVote appends a participant and increments the tally. It refuses
// invalid states: not open, expired, already voted.
func (v *Vote) CastVote(adminUserID int64, choice Choice, now time.Time) error {
	if v.Status != StatusOpen {
		return ErrVoteNotOpen
	}
	if now.After(v.ExpiresAt) {
		return ErrVoteExpired
	}
	for _, p := range v.Participants {
		if p.AdminUserID == adminUserID {
			return ErrAlreadyVoted
		}
	}

	v.Participants = append(v.Participants, Participant{
		AdminUserID: adminUserID,
		Vote:        choice,
		VotedAt:     now,
	})
	if choice == ChoiceFor {
		v.VotesFor++
	} else {
		v.VotesAgainst++
	}
	v.maybeResolve(now)
	return nil
}

// Expire transitions an Open vote to Expired if past its deadline.
// Called by the sweeper, not by request handlers.
func (v *Vote) Expire(now time.Time) error {
	if v.Status != StatusOpen {
		return ErrVoteNotOpen
	}
	if now.Before(v.ExpiresAt) {
		return ErrNotYetExpired
	}
	v.Status = StatusExpired
	v.ResolvedAt = &now
	return nil
}

// Resolution returns the outcome once resolved. nil while open.
func (v *Vote) Resolution() *Choice {
	if v.Status != StatusResolved {
		return nil
	}
	if v.VotesFor >= v.RequiredVotes {
		c := ChoiceFor
		return &c
	}
	if v.VotesAgainst >= v.RequiredVotes {
		c := ChoiceAgainst
		return &c
	}
	return nil
}

func (v *Vote) maybeResolve(now time.Time) {
	if v.Status != StatusOpen {
		return
	}
	if v.VotesFor >= v.RequiredVotes || v.VotesAgainst >= v.RequiredVotes {
		v.Status = StatusResolved
		v.ResolvedAt = &now
	}
}
