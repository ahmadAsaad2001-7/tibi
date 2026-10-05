package listnotifications

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/communication/listnotifications/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type NotificationOut struct {
	ID        int64          `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Payload   map[string]any `json:"payload,omitempty"`
	IsRead    bool           `json:"is_read"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Response struct {
	Items   []NotificationOut `json:"items"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
	Total   int64             `json:"total"`
	Unread  int64             `json:"unread"`
}

func (s *Service) Execute(ctx context.Context, userID int64, page, perPage int) (*Response, error) {
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

		rows, err := q.ListNotifications(ctx, db.ListNotificationsParams{
			UserID: userID,
			Limit:  int32(perPage),
			Offset: int32(offset),
		})
		if err != nil {
			return httpx.Internal(err)
		}

		total, err := q.CountNotifications(ctx, userID)
		if err != nil {
			return httpx.Internal(err)
		}

		items := make([]NotificationOut, len(rows))
		for i, r := range rows {
			items[i] = NotificationOut{
				ID:        r.ID,
				Type:      r.Type,
				Title:     r.Title,
				Body:      r.Body,
				Payload:   parsePayload(r.Payload),
				IsRead:    r.IsRead,
				ReadAt:    tsPtr(r.ReadAt),
				CreatedAt: r.CreatedAt.Time,
			}
		}
		unread := int64(0)
		for _, it := range items {
			if !it.IsRead {
				unread++
			}
		}
		resp = &Response{
			Items:   items,
			Page:    page,
			PerPage: perPage,
			Total:   total,
			Unread:  unread,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func parsePayload(b []byte) map[string]any {
	if len(b) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
