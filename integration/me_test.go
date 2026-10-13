//go:build integration

package integration

import (
	"context"
	"testing"

	filescontracts "tibi/internal/files/contracts"
	"tibi/internal/identity/me"
	"tibi/internal/identity/register"
)

// TestMe_PatientRow_ScansNullableDoctorColumns is a regression test for a bug
// found by the slice-18 endpoint sweep: GetMe LEFT JOINs doctors_profiles,
// whose NOT NULL consultation_fee / average_rating columns come back NULL for
// patient rows. sqlc types them as plain string, so pgx failed scanning every
// patient's /me (500 internal_error: cannot scan NULL into *string). The fix
// wraps those two columns in CASE WHEN d.id IS NULL in query.sql.
func TestMe_PatientRow_ScansNullableDoctorColumns(t *testing.T) {
	db := newDB(t)
	svc := me.NewService(db, stubFiles{})

	resp, err := newRegisterService(db).Execute(context.Background(), register.Command{
		Email:       "me-patient@example.com",
		Password:    "correct-horse-battery-staple",
		Role:        "Patient",
		FullName:    "Me Patient",
		PhoneNumber: "+201234567891",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := svc.Execute(context.Background(), resp.User.ID)
	if err != nil {
		t.Fatalf("/me for patient must not fail on NULL doctor columns: %v", err)
	}
	if got.PatientProfile == nil {
		t.Fatal("expected patient_profile to be populated")
	}
	if got.DoctorProfile != nil {
		t.Fatal("patient must not get a doctor_profile")
	}
	if got.PatientProfile.FullName != "Me Patient" {
		t.Fatalf("full_name = %q", got.PatientProfile.FullName)
	}
}

// TestMe_DoctorRow_PreservesProfileFields proves the CASE-based fix for the
// patient NULL scan does not distort real doctor values.
func TestMe_DoctorRow_PreservesProfileFields(t *testing.T) {
	db := newDB(t)
	svc := me.NewService(db, stubFiles{})

	resp, err := newRegisterService(db).Execute(context.Background(), register.Command{
		Email:       "me-doctor@example.com",
		Password:    "correct-horse-battery-staple",
		Role:        "Doctor",
		FullName:    "Me Doctor",
		PhoneNumber: "+201234567892",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := svc.Execute(context.Background(), resp.User.ID)
	if err != nil {
		t.Fatalf("/me for doctor: %v", err)
	}
	if got.DoctorProfile == nil {
		t.Fatal("expected doctor_profile to be populated")
	}
	if got.PatientProfile != nil {
		t.Fatal("doctor must not get a patient_profile")
	}
	// consultation_fee defaults to 0 and is rendered as text by GetMe.
	if got.DoctorProfile.ConsultationFee != "0.00" {
		t.Fatalf("consultation_fee = %q, want 0.00", got.DoctorProfile.ConsultationFee)
	}
	if got.DoctorProfile.AverageRating != "0.00" {
		t.Fatalf("average_rating = %q, want 0.00", got.DoctorProfile.AverageRating)
	}
}

// stubFiles is a minimal filescontracts.API; neither test has a
// profile_image_file_id, so PresignGet is never reached.
type stubFiles struct{}

func (stubFiles) Get(context.Context, int64) (*filescontracts.FileMetadata, error) {
	return nil, filescontracts.ErrNotFound
}

func (stubFiles) PresignGet(context.Context, int64, int) (string, error) {
	return "", filescontracts.ErrNotFound
}