package doctorprofile

import (
	"strconv"
	"time"
)

type VerificationStatus string

const (
	StatusNotSubmitted  VerificationStatus = "NotSubmitted"
	StatusPendingReview VerificationStatus = "PendingReview"
	StatusVerified      VerificationStatus = "Verified"
	StatusRejected      VerificationStatus = "Rejected"
)

type Profile struct {
	ID                          int64
	UserID                      int64
	FullName                    string
	Bio                         string
	ConsultationFee             string // decimal as string, e.g. "250.00"
	Currency                    string
	ClinicName                  string
	ClinicAddress               string
	MedicalLicenseNumber        *string
	VerificationStatus          VerificationStatus
	VerificationRejectionReason *string
	SubmittedAt                 *time.Time
	AverageRating               string
	RatingCount                 int
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
	DeletedAt                   *time.Time

	// Loaded eagerly with the aggregate. Not a navigation; a set of IDs
	// owned by this aggregate's transaction.
	SpecialtyIDs []int64
}

// SubmitForVerification is the state transition from draft to review.
// Rejected profiles may resubmit after editing; verified ones cannot.
func (p *Profile) SubmitForVerification(now time.Time) error {
	switch p.VerificationStatus {
	case StatusVerified:
		return ErrAlreadyVerified
	case StatusPendingReview:
		return ErrAlreadyPendingReview
	}

	if missing := p.missingForSubmission(); len(missing) > 0 {
		return &IncompleteProfileError{Missing: missing}
	}

	p.VerificationStatus = StatusPendingReview
	p.SubmittedAt = &now
	p.VerificationRejectionReason = nil
	return nil
}

// UpdateFields is a domain method, not a setter. It applies only
// the fields the caller asked to change.
func (p *Profile) UpdateFields(in UpdateInput) {
	if in.Bio != nil {
		p.Bio = *in.Bio
	}
	if in.ConsultationFee != nil {
		p.ConsultationFee = *in.ConsultationFee
	}
	if in.Currency != nil {
		p.Currency = *in.Currency
	}
	if in.ClinicName != nil {
		p.ClinicName = *in.ClinicName
	}
	if in.ClinicAddress != nil {
		p.ClinicAddress = *in.ClinicAddress
	}
	if in.MedicalLicenseNumber != nil {
		p.MedicalLicenseNumber = in.MedicalLicenseNumber
	}
	if in.SpecialtyIDs != nil {
		p.SpecialtyIDs = *in.SpecialtyIDs
	}
}

type UpdateInput struct {
	Bio                  *string
	ConsultationFee      *string
	Currency             *string
	ClinicName           *string
	ClinicAddress        *string
	MedicalLicenseNumber *string
	SpecialtyIDs         *[]int64
}

func (p *Profile) missingForSubmission() []string {
	var missing []string
	if p.Bio == "" {
		missing = append(missing, "bio")
	}
	if p.ClinicName == "" {
		missing = append(missing, "clinic_name")
	}
	if p.ClinicAddress == "" {
		missing = append(missing, "clinic_address")
	}
	if p.MedicalLicenseNumber == nil || *p.MedicalLicenseNumber == "" {
		missing = append(missing, "medical_license_number")
	}
	if !feeIsPositive(p.ConsultationFee) {
		missing = append(missing, "consultation_fee")
	}
	if len(p.SpecialtyIDs) == 0 {
		missing = append(missing, "specialty_ids")
	}
	return missing
}

func feeIsPositive(s string) bool {
	f, err := strconv.ParseFloat(s, 64)
	return err == nil && f > 0
}