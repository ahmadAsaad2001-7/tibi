package listvotes

import (
	"context"
	"time"

	"tibi/internal/admin/listvotes/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	VoteID         int64     `json:"vote_id"`
	ActionType     string    `json:"action_type"`
	TargetUserID   int64     `json:"target_user_id"`
	TargetEmail    string    `json:"target_email"`
	TargetRole     string    `json:"target_role"`
	TargetFullName string    `json:"target_full_name"`
	VotesFor       int       `json:"votes_for"`
	VotesAgainst   int       `json:"votes_against"`
	RequiredVotes  int       `json:"required_votes"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type Response struct {
	Items []Item `json:"items"`
}

func (s *Service) Execute(ctx context.Context, adminUserID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	rows, err := q.ListOpenVotes(ctx)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, Item{
			VoteID:         r.ID,
			ActionType:     r.ActionType,
			TargetUserID:   r.TargetUserID,
			TargetEmail:    r.TargetEmail,
			TargetRole:     string(r.TargetRole),
			TargetFullName: r.TargetFullName,
			VotesFor:       int(r.VotesFor),
			VotesAgainst:   int(r.VotesAgainst),
			RequiredVotes:  int(r.RequiredVotes),
			ExpiresAt:      r.ExpiresAt.Time,
			CreatedAt:      r.CreatedAt.Time,
		})
	}
	_ = adminUserID // reserved for slice 4: filter to votes this admin has not voted on
	return &Response{Items: items}, nil
}
