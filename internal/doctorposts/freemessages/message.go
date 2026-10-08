package freemessages

import (
	"errors"
	"strings"
	"time"
)

const (
	MaxNameLength    = 200
	MaxEmailLength   = 320
	MaxContentLength = 4000
	MaxReplyLength   = 4000
)

var (
	ErrAlreadyReplied = errors.New("message is already replied to")
	ErrEmptyContent   = errors.New("content is empty")
	ErrTooLong        = errors.New("content exceeds maximum length")
	ErrEmptyName      = errors.New("sender name is empty")
)

type Message struct {
	ID              int64
	DoctorProfileID int64
	SenderName      string
	SenderEmail     string
	SenderPhone     *string
	Content         string
	IsRepliedTo     bool
	RepliedAt       *time.Time
	ReplyContent    *string
	SenderIP        *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// New validates and constructs a message. The only way to create one.
func New(doctorProfileID int64, name, email string, phone *string, content, senderIP string, now time.Time) (*Message, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	if len([]rune(name)) > MaxNameLength {
		return nil, ErrTooLong
	}
	email = strings.TrimSpace(email)
	if len(email) < 3 || len(email) > MaxEmailLength {
		return nil, errors.New("invalid email length")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyContent
	}
	if len([]rune(content)) > MaxContentLength {
		return nil, ErrTooLong
	}

	var ip *string
	if senderIP != "" {
		ip = &senderIP
	}

	return &Message{
		DoctorProfileID: doctorProfileID,
		SenderName:      name,
		SenderEmail:     email,
		SenderPhone:     phone,
		Content:         content,
		SenderIP:        ip,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// MarkReplied enforces the one-reply invariant.
func (m *Message) MarkReplied(replyContent string, now time.Time) error {
	if m.IsRepliedTo {
		return ErrAlreadyReplied
	}
	replyContent = strings.TrimSpace(replyContent)
	if replyContent == "" {
		return ErrEmptyContent
	}
	if len([]rune(replyContent)) > MaxReplyLength {
		return ErrTooLong
	}
	m.IsRepliedTo = true
	m.RepliedAt = &now
	m.ReplyContent = &replyContent
	m.UpdatedAt = now
	return nil
}
