package contracts

import (
	"context"
	"time"
)

// API is the only surface other modules may import from Communication.
// It has two responsibilities: sending messages on behalf of a module (rare;
// messages are usually sent by HTTP handlers), and creating notifications
// (common; every module creates them).
type API interface {
	// CreateNotification inserts a notification row on the caller's
	// transaction. The caller is responsible for publishing
	// `NotificationCreated` to the hub after commit.
	CreateNotification(ctx context.Context, in CreateNotificationInput) (int64, error)

	// NotifyMessage is a convenience for slices that create a message and
	// want the notification row + WS event to follow. It does both.
	NotifyMessage(ctx context.Context, in NotifyMessageInput) error
}

type CreateNotificationInput struct {
	UserID  int64
	Type    string
	Title   string
	Body    string
	Payload map[string]any
}

type NotifyMessageInput struct {
	MessageID       int64
	ConsultationID  int64
	RecipientUserID int64
	SenderUserID    int64
	Preview         string
	SentAt          time.Time
}
