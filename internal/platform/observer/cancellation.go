package observer

import "context"

// CancellationObserver is notified after a consultation is cancelled.
// Implemented by modules that keep state tied to a consultation's lifecycle
// (Queue, Clinical, etc.). Consultations defines the call site; the
// implementing modules define the behavior.
type CancellationObserver interface {
	OnConsultationCancelled(ctx context.Context, consultationID int64) error
}
