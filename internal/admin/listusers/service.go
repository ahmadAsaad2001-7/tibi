package listusers

import (
	"context"
	"time"

	"tibi/internal/admin/listusers/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type Response struct {
	Items []Item `json:"items"`
	Total int64  `json:"total"`
}

func (s *Service) Execute(ctx context.Context, role, search *string, limit, offset int32) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	rows, err := q.ListUsers(ctx, db.ListUsersParams{
		Limit:  limit,
		Offset: offset,
		Role:   role,
		Search: search,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	total, err := q.CountUsers(ctx, db.CountUsersParams{Role: role, Search: search})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, Item{
			ID:            r.ID,
			Email:         r.Email,
			Role:          string(r.Role),
			EmailVerified: r.EmailVerifiedAt.Valid,
			CreatedAt:     r.CreatedAt.Time,
		})
	}
	return &Response{Items: items, Total: total}, nil
}