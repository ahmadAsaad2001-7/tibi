package notifymessage

import (
	"context"

	"tibi/internal/communication/contracts"
	"tibi/internal/communication/createnotification"
)

type Service struct {
	creator *createnotification.Service
}

func NewService(creator *createnotification.Service) *Service {
	return &Service{creator: creator}
}

// Execute creates the MessageReceived notification row inside the caller's
// transaction. It does NOT publish to the hub; the caller (sendmessage) is
// responsible for that after commit, so a rolled-back create never fans out.
func (s *Service) Execute(ctx context.Context, in contracts.NotifyMessageInput) error {
	_, err := s.creator.Create(ctx, contracts.CreateNotificationInput{
		UserID: in.RecipientUserID,
		Type:   "MessageReceived",
		Title:  "New message",
		Body:   in.Preview,
		Payload: map[string]any{
			"consultation_id": in.ConsultationID,
			"message_id":      in.MessageID,
		},
	})
	return err
}
