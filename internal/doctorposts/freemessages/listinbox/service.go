package listinbox

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctorposts/freemessages/listinbox/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	ID           int64      `json:"id"`
	SenderName   string     `json:"sender_name"`
	SenderEmail  string     `json:"sender_email"`
	SenderPhone  *string    `json:"sender_phone,omitempty"`
	Content      string     `json:"content"`
	IsRepliedTo  bool       `json:"is_replied_to"`
	RepliedAt    *time.Time `json:"replied_at,omitempty"`
	ReplyContent *string    `json:"reply_content,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Response struct {
	Items      []Item `json:"items"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"total_pages"`
}

func (s *Service) Execute(ctx context.Context, userID int64, unrepliedOnly bool, page, perPage int) (*Response, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	q := db.New(s.db.Querier(ctx))

	profileID, err := q.GetDoctorProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}

	rows, err := q.ListInbox(ctx, db.ListInboxParams{
		DoctorProfileID: profileID,
		UnrepliedOnly:   &unrepliedOnly,
		Limit:           int32(perPage),
		Offset:          int32(offset),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	total, err := q.CountInbox(ctx, db.CountInboxParams{
		DoctorProfileID: profileID,
		UnrepliedOnly:   &unrepliedOnly,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	items := make([]Item, len(rows))
	for i, r := range rows {
		items[i] = Item{
			ID:           r.ID,
			SenderName:   r.SenderName,
			SenderEmail:  r.SenderEmail,
			SenderPhone:  r.SenderPhone,
			Content:      r.Content,
			IsRepliedTo:  r.IsRepliedTo,
			RepliedAt:    tsPtr(r.RepliedAt),
			ReplyContent: r.ReplyContent,
			CreatedAt:    r.CreatedAt.Time,
		}
	}

	return &Response{
		Items:      items,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: int((total + int64(perPage) - 1) / int64(perPage)),
	}, nil
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
