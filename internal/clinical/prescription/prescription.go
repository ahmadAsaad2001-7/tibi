package prescription

import (
	"errors"
	"time"
)

type Prescription struct {
	ID               int64
	ConsultationID   int64
	PatientProfileID int64
	DoctorProfileID  int64
	IssuedAt         time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time

	Medications []Medication
}

type Medication struct {
	ID             int64
	PrescriptionID int64
	Name           string
	Dosage         string
	Frequency      string
	DurationDays   int
	Notes          *string
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

var (
	ErrEmpty = errors.New("prescription must have at least one medication")
)

// ReplaceMedications replaces the full set in one call. Called on upsert.
// The domain method enforces the "at least one medication" invariant, which
// the SQL layer cannot express cleanly across a delete + insert.
func (p *Prescription) ReplaceMedications(meds []Medication, now time.Time) error {
	if len(meds) == 0 {
		return ErrEmpty
	}
	for i := range meds {
		meds[i].PrescriptionID = p.ID
		meds[i].CreatedAt = now
	}
	p.Medications = meds
	p.UpdatedAt = now
	return nil
}
