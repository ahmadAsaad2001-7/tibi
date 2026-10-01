package doctorprofile

import (
	"errors"
	"testing"
	"time"
)

func completeProfile() *Profile {
	license := "LIC-1"
	return &Profile{
		FullName:             "Dr Ada",
		Bio:                  "Cardiology",
		ConsultationFee:      "250.00",
		Currency:             "EGP",
		ClinicName:           "Nile Clinic",
		ClinicAddress:        "12 Corniche",
		MedicalLicenseNumber: &license,
		VerificationStatus:   StatusNotSubmitted,
		SpecialtyIDs:         []int64{1},
	}
}

func TestSubmitForVerification(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	p := completeProfile()
	reason := "missing license"
	p.VerificationRejectionReason = &reason

	if err := p.SubmitForVerification(now); err != nil {
		t.Fatal(err)
	}
	if p.VerificationStatus != StatusPendingReview {
		t.Fatalf("status = %s", p.VerificationStatus)
	}
	if p.SubmittedAt == nil || !p.SubmittedAt.Equal(now) {
		t.Fatalf("submitted_at = %v", p.SubmittedAt)
	}
	if p.VerificationRejectionReason != nil {
		t.Fatal("rejection reason should be cleared")
	}
}

func TestSubmitIncompleteProfile(t *testing.T) {
	p := &Profile{VerificationStatus: StatusRejected}
	err := p.SubmitForVerification(time.Now())
	var incomplete *IncompleteProfileError
	if !errors.As(err, &incomplete) {
		t.Fatalf("err = %v, want IncompleteProfileError", err)
	}
	fields := incomplete.Fields()
	for _, name := range []string{"bio", "clinic_name", "clinic_address", "medical_license_number", "consultation_fee", "specialty_ids"} {
		if fields[name] == "" {
			t.Fatalf("missing field %s in %v", name, fields)
		}
	}
	empty := ""
	p.MedicalLicenseNumber = &empty
	p.ConsultationFee = "0"
	if err := p.SubmitForVerification(time.Now()); !errors.As(err, &incomplete) {
		t.Fatalf("blank license and zero fee should stay incomplete, err=%v", err)
	}
}

func TestSubmitTerminalStatuses(t *testing.T) {
	verified := completeProfile()
	verified.VerificationStatus = StatusVerified
	if err := verified.SubmitForVerification(time.Now()); !errors.Is(err, ErrAlreadyVerified) {
		t.Fatalf("err = %v", err)
	}

	pending := completeProfile()
	pending.VerificationStatus = StatusPendingReview
	if err := pending.SubmitForVerification(time.Now()); !errors.Is(err, ErrAlreadyPendingReview) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateFieldsTouchesOnlyProvidedValues(t *testing.T) {
	p := completeProfile()
	bio := "Updated bio"
	ids := []int64{4, 5}
	p.UpdateFields(UpdateInput{Bio: &bio, SpecialtyIDs: &ids})

	if p.Bio != bio || p.ClinicName != "Nile Clinic" || p.ConsultationFee != "250.00" {
		t.Fatalf("unexpected profile after partial update: %+v", p)
	}
	if len(p.SpecialtyIDs) != 2 || p.SpecialtyIDs[0] != 4 {
		t.Fatalf("specialty ids = %v", p.SpecialtyIDs)
	}
}
