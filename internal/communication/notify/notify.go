package notify

import (
	"context"
	"time"

	"tibi/internal/communication/contracts"
)

// QueueCalled notifies the patient that the doctor is ready.
func QueueCalled(ctx context.Context, api contracts.API, recipientUserID, entryID int64, queueNumber int) error {
	_, err := api.CreateNotification(ctx, contracts.CreateNotificationInput{
		UserID: recipientUserID,
		Type:   "QueueCalled",
		Title:  "It's your turn",
		Body:   "The doctor is ready for you.",
		Payload: map[string]any{
			"entry_id":     entryID,
			"queue_number": queueNumber,
		},
	})
	return err
}

// ConsultationConfirmed notifies the patient that the payment cleared.
func ConsultationConfirmed(ctx context.Context, api contracts.API, recipientUserID, consultationID int64) error {
	_, err := api.CreateNotification(ctx, contracts.CreateNotificationInput{
		UserID: recipientUserID,
		Type:   "ConsultationConfirmed",
		Title:  "Consultation confirmed",
		Body:   "Your consultation is confirmed.",
		Payload: map[string]any{
			"consultation_id": consultationID,
		},
	})
	return err
}

// ConsultationBooked notifies the doctor of a new booking.
func ConsultationBooked(ctx context.Context, api contracts.API, recipientUserID, consultationID int64, scheduledAt time.Time) error {
	_, err := api.CreateNotification(ctx, contracts.CreateNotificationInput{
		UserID: recipientUserID,
		Type:   "ConsultationBooked",
		Title:  "New booking",
		Body:   "A patient booked a consultation.",
		Payload: map[string]any{
			"consultation_id": consultationID,
			"scheduled_at":    scheduledAt.Format(time.RFC3339),
		},
	})
	return err
}
