package listmessages

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/communication/listmessages/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type MessageOut struct {
	ID           int64      `json:"id"`
	SenderUserID int64      `json:"sender_user_id"`
	Content      string     `json:"content"`
	SentAt       time.Time  `json:"sent_at"`
	ReadAt       *time.Time `json:"read_at,omitempty"`
}

type Response struct {
	Items   []MessageOut `json:"items"`
	Page    int          `json:"page"`
	PerPage int          `json:"per_page"`
	Total   int64        `json:"total"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64, page, perPage int) (*Response, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	var resp *Response
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		rows, err := q.ListMessages(ctx, db.ListMessagesParams{
			ConsultationID: consultationID,
			UserID:         userID,
			Limit:          int32(perPage),
			Offset:         int32(offset),
		})
		if err != nil {
			return httpx.Internal(err)
		}

		total, err := q.CountMessages(ctx, db.CountMessagesParams{
			ConsultationID: consultationID,
			UserID:         userID,
		})
		if err != nil {
			return httpx.Internal(err)
		}

		// Side effect: marking messages read is the read event. Runs on the
		// same transaction so it commits atomically with the read.
		if err := q.MarkMessagesRead(ctx, db.MarkMessagesReadParams{
			ConsultationID: consultationID,
			SenderUserID:   userID,
		}); err != nil {
			return httpx.Internal(err)
		}

		items := make([]MessageOut, len(rows))
		for i, r := range rows {
			items[i] = MessageOut{
				ID:           r.ID,
				SenderUserID: r.SenderUserID,
				Content:      r.Content,
				SentAt:       r.SentAt.Time,
				ReadAt:       tsPtr(r.ReadAt),
			}
		}
		resp = &Response{
			Items:   items,
			Page:    page,
			PerPage: perPage,
			Total:   total,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
