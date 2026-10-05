package notification

import "time"

// Type is a string. Same reasoning as AdminVote.action_type: adding a
// notification type is a Go constant, not a schema change.
const (
	TypeConsultationBooked    = "ConsultationBooked"
	TypeConsultationConfirmed = "ConsultationConfirmed"
	TypeConsultationCancelled = "ConsultationCancelled"
	TypeQueueCalled           = "QueueCalled"
	TypeMessageReceived       = "MessageReceived"
	TypeMedicalRecordUpdated  = "MedicalRecordUpdated"
	TypeDoctorVerified        = "DoctorVerified"
	TypeDoctorRejected        = "DoctorRejected"
)

type Notification struct {
	ID        int64
	UserID    int64
	Type      string
	Title     string
	Body      string
	Payload   map[string]any
	IsRead    bool
	ReadAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (n *Notification) MarkRead(now time.Time) {
	if !n.IsRead {
		n.IsRead = true
		n.ReadAt = &now
		n.UpdatedAt = now
	}
}
