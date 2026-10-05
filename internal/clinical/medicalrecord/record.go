package medicalrecord

import (
	"errors"
	"time"
)

type Record struct {
	ID                 int64
	ConsultationID     int64
	PatientProfileID   int64
	DoctorProfileID    int64
	Allergies          *string
	CurrentMedications *string
	PastConditions     *string
	DoctorNotes        *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time

	// Loaded with the aggregate on write paths. Reads bypass.
	Attachments []Attachment
}

type Attachment struct {
	ID              int64
	MedicalRecordID int64
	FileID          int64
	Label           *string
	CreatedAt       time.Time
	DeletedAt       *time.Time
}

var (
	ErrAlreadyDeleted    = errors.New("record is deleted")
	ErrAttachmentMissing = errors.New("attachment not found in this record")
)

// Update applies the mutable fields. A nil **string means "do not change".
// A non-nil **string whose inner value is nil (or points to ""?) — the
// caller passes **string so that "omit" (nil outer) vs "clear" (outer set,
// inner nil or empty) can be distinguished.
func (r *Record) Update(in UpdateInput, now time.Time) {
	if in.Allergies != nil {
		r.Allergies = *in.Allergies
	}
	if in.CurrentMedications != nil {
		r.CurrentMedications = *in.CurrentMedications
	}
	if in.PastConditions != nil {
		r.PastConditions = *in.PastConditions
	}
	if in.DoctorNotes != nil {
		r.DoctorNotes = *in.DoctorNotes
	}
	r.UpdatedAt = now
}

type UpdateInput struct {
	Allergies          **string
	CurrentMedications **string
	PastConditions     **string
	DoctorNotes        **string
}

// AddAttachment is a domain method so the invariant "attachment file ids
// are unique within a record" lives in the aggregate.
func (r *Record) AddAttachment(fileID int64, label *string, now time.Time) (*Attachment, error) {
	for _, a := range r.Attachments {
		if a.DeletedAt == nil && a.FileID == fileID {
			return nil, errors.New("file already attached")
		}
	}
	a := Attachment{
		MedicalRecordID: r.ID,
		FileID:          fileID,
		Label:           label,
		CreatedAt:       now,
	}
	r.Attachments = append(r.Attachments, a)
	return &a, nil
}

// RemoveAttachment marks a child attachment as deleted. Returns an error if
// the attachment id does not belong to this record.
func (r *Record) RemoveAttachment(attachmentID int64, now time.Time) error {
	for i := range r.Attachments {
		if r.Attachments[i].ID == attachmentID && r.Attachments[i].DeletedAt == nil {
			r.Attachments[i].DeletedAt = &now
			return nil
		}
	}
	return ErrAttachmentMissing
}
