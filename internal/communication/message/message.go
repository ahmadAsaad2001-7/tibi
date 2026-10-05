package message

import (
	"errors"
	"strings"
	"time"
)

const MaxContentLength = 4000

type Message struct {
	ID             int64
	ConsultationID int64
	SenderUserID   int64
	Content        string
	SentAt         time.Time
	ReadAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

var (
	ErrEmpty   = errors.New("message content is empty")
	ErrTooLong = errors.New("message content exceeds 4000 characters")
)

// New constructs a message with validation. The only way to create one.
func New(consultationID, senderUserID int64, content string, now time.Time) (*Message, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, ErrEmpty
	}
	if len([]rune(trimmed)) > MaxContentLength {
		return nil, ErrTooLong
	}
	return &Message{
		ConsultationID: consultationID,
		SenderUserID:   senderUserID,
		Content:        trimmed,
		SentAt:         now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (m *Message) MarkRead(now time.Time) {
	if m.ReadAt == nil {
		m.ReadAt = &now
		m.UpdatedAt = now
	}
}
