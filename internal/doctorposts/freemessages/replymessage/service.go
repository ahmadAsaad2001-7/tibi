package replymessage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctorposts/freemessages"
	"tibi/internal/doctorposts/freemessages/replymessage/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/email"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db     *database.DB
	mailer email.Sender
	clock  func() time.Time
}

func NewService(db *database.DB, mailer email.Sender) *Service {
	return &Service{db: db, mailer: mailer, clock: time.Now}
}

type Response struct {
	ID          int64      `json:"id"`
	IsRepliedTo bool       `json:"is_replied_to"`
	RepliedAt   *time.Time `json:"replied_at,omitempty"`
	EmailSent   bool       `json:"email_sent"`
}

func (s *Service) Execute(ctx context.Context, userID, messageID int64, cmd Command) (*Response, error) {
	var resp *Response
	var recipientEmail, originalContent string

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		profileID, err := q.GetDoctorProfileIDForUser(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("doctor profile not found")
			}
			return httpx.Internal(err)
		}

		row, err := q.GetFreeMessageForReply(ctx, messageID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("message not found")
			}
			return httpx.Internal(err)
		}

		// Anti-enumeration: doctor may only access their own inbox.
		if row.DoctorProfileID != profileID {
			return httpx.NotFound("message not found")
		}

		msg := &freemessages.Message{
			ID:              row.ID,
			DoctorProfileID: row.DoctorProfileID,
			SenderName:      row.SenderName,
			SenderEmail:     row.SenderEmail,
			Content:         row.Content,
			IsRepliedTo:     row.IsRepliedTo,
			RepliedAt:       tsPtr(row.RepliedAt),
			ReplyContent:    row.ReplyContent,
		}

		now := s.clock()
		if err := msg.MarkReplied(cmd.Content, now); err != nil {
			if errors.Is(err, freemessages.ErrAlreadyReplied) {
				return httpx.Conflict("message is already replied to")
			}
			return httpx.ValidationFailed(map[string]string{"content": err.Error()})
		}

		n, err := q.UpdateReply(ctx, db.UpdateReplyParams{
			ID:           msg.ID,
			RepliedAt:    pgTime(now),
			ReplyContent: msg.ReplyContent,
			Xmin:         pgXmin(row.Xmin),
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("message was modified concurrently")
		}

		recipientEmail = msg.SenderEmail
		originalContent = msg.Content
		t := now
		resp = &Response{
			ID:          msg.ID,
			IsRepliedTo: true,
			RepliedAt:   &t,
			EmailSent:   false,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// After commit: best-effort email. SD28.
	if s.mailer != nil {
		body := buildEmailBody(recipientEmail, originalContent, cmd.Content)
		if err := s.mailer.Send(ctx, recipientEmail, "Reply from your doctor", body); err != nil {
			// Log, do not fail the request. The reply is stored.
			resp.EmailSent = false
		} else {
			resp.EmailSent = true
		}
	}

	return resp, nil
}

func buildEmailBody(to, original, reply string) string {
	return "Your message:\n" + original + "\n\nReply:\n" + reply
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
