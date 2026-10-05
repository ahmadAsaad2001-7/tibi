package sendmessage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/communication/contracts"
	"tibi/internal/communication/message"
	"tibi/internal/communication/sendmessage/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
)

type Service struct {
	db    *database.DB
	hub   ws.Hub
	notif contracts.API
	clock func() time.Time
}

func NewService(db *database.DB, hub ws.Hub, notif contracts.API) *Service {
	return &Service{db: db, hub: hub, notif: notif, clock: time.Now}
}

type Response struct {
	ID             int64     `json:"id"`
	ConsultationID int64     `json:"consultation_id"`
	SenderUserID   int64     `json:"sender_user_id"`
	Content        string    `json:"content"`
	SentAt         time.Time `json:"sent_at"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64, cmd Command) (*Response, error) {
	msg, err := message.New(consultationID, userID, cmd.Content, s.clock())
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"content": err.Error()})
	}

	var resp *Response
	var recipientID int64

	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		c, err := q.GetConsultationParticipants(ctx, consultationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("consultation not found")
			}
			return httpx.Internal(err)
		}

		// Caller must be one of the two participants.
		switch userID {
		case c.PatientUserID:
			recipientID = c.DoctorUserID
		case c.DoctorUserID:
			recipientID = c.PatientUserID
		default:
			return httpx.NotFound("consultation not found")
		}

		row, err := q.InsertMessage(ctx, db.InsertMessageParams{
			ConsultationID: consultationID,
			SenderUserID:   userID,
			Content:        msg.Content,
			SentAt:         pgTime(msg.SentAt),
		})
		if err != nil {
			return httpx.Internal(err)
		}
		msg.ID = row.ID

		// Notification row inside the same transaction. If this fails,
		// the message insert rolls back too.
		if err := s.notif.NotifyMessage(ctx, contracts.NotifyMessageInput{
			MessageID:       msg.ID,
			ConsultationID:  consultationID,
			RecipientUserID: recipientID,
			SenderUserID:    userID,
			Preview:         preview(msg.Content),
			SentAt:          msg.SentAt,
		}); err != nil {
			return err
		}

		resp = &Response{
			ID:             msg.ID,
			ConsultationID: consultationID,
			SenderUserID:   userID,
			Content:        msg.Content,
			SentAt:         msg.SentAt,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// After commit: WS fan-out, run on the post-commit hook so it is not tied
	// to the (possibly cancelled) request context. Fire-and-forget.
	s.db.AfterCommit(ctx, func() {
		if s.hub != nil {
			s.hub.SendToConsultation(consultationID, ws.Event{
				Type:    "MessageReceived",
				Payload: resp,
			})
		}
	})
	return resp, nil
}

// preview truncates content on rune boundaries (not bytes) so the
// notification body never carries trailing invalid UTF-8.
func preview(s string) string {
	r := []rune(s)
	if len(r) <= 80 {
		return s
	}
	return string(r[:77]) + "..."
}
