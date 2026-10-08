package listsuspensions

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/admin/listsuspensions/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	UserEmail  string     `json:"user_email"`
	UserRole   string     `json:"user_role"`
	Reason     string     `json:"reason"`
	FromTS     time.Time  `json:"from_ts"`
	ToTS       *time.Time `json:"to_ts,omitempty"`
	LiftedAt   *time.Time `json:"lifted_at,omitempty"`
	LiftedBy   *int64     `json:"lifted_by,omitempty"`
	LiftReason *string    `json:"lift_reason,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Response struct {
	Items []Item `json:"items"`
	Total int64  `json:"total"`
}

func (s *Service) Execute(ctx context.Context, activeOnly *bool, limit, offset int32) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	rows, err := q.ListSuspensions(ctx, db.ListSuspensionsParams{
		Limit:      limit,
		Offset:     offset,
		ActiveOnly: activeOnly,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	total, err := q.CountSuspensions(ctx, activeOnly)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	items := make([]Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, Item{
			ID:         r.ID,
			UserID:     r.UserID,
			UserEmail:  r.UserEmail,
			UserRole:   string(r.UserRole),
			Reason:     r.Reason,
			FromTS:     r.FromTs.Time,
			ToTS:       tsPtr(r.ToTs),
			LiftedAt:   tsPtr(r.LiftedAt),
			LiftedBy:   r.LiftedBy,
			LiftReason: r.LiftReason,
			CreatedAt:  r.CreatedAt.Time,
		})
	}
	return &Response{Items: items, Total: total}, nil
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}