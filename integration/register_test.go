//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"tibi/internal/doctors/api"
	"tibi/internal/doctors/createsession"
	"tibi/internal/identity/register"
	patientcontracts "tibi/internal/patients/contracts"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

// TestRegister_CreatesUserAndPatientProfile proves the register transaction
// writes both identity_users and patients_profiles atomically.
func TestRegister_CreatesUserAndPatientProfile(t *testing.T) {
	db := newDB(t)
	svc := newRegisterService(db)

	resp, err := svc.Execute(context.Background(), register.Command{
		Email:       "integration@example.com",
		Password:    "correct-horse-battery-staple",
		Role:        "Patient",
		FullName:    "Integration Test",
		PhoneNumber: "+201234567890",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.User.ID == 0 {
		t.Fatal("expected a real user id")
	}
	if resp.User.Role != "Patient" {
		t.Fatalf("role = %q, want Patient", resp.User.Role)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	var profileCount int
	err = db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM patients_profiles WHERE user_id = $1`,
		resp.User.ID).Scan(&profileCount)
	if err != nil {
		t.Fatal(err)
	}
	if profileCount != 1 {
		t.Fatalf("patients_profiles = %d rows, want 1", profileCount)
	}

	// The transaction must also have issued a refresh token for the session.
	var tokenCount int
	err = db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM identity_refresh_tokens WHERE user_id = $1`,
		resp.User.ID).Scan(&tokenCount)
	if err != nil {
		t.Fatal(err)
	}
	if tokenCount != 1 {
		t.Fatalf("identity_refresh_tokens = %d rows, want 1", tokenCount)
	}
}

// TestRegister_DuplicateEmail proves the UNIQUE(email) constraint race is
// surfaced as "already_exists", not a 500.
func TestRegister_DuplicateEmail(t *testing.T) {
	db := newDB(t)
	svc := newRegisterService(db)

	cmd := register.Command{
		Email:       "dup@example.com",
		Password:    "correct-horse-battery-staple",
		Role:        "Patient",
		FullName:    "Dup User",
		PhoneNumber: "+201234567890",
	}
	if _, err := svc.Execute(context.Background(), cmd); err != nil {
		t.Fatalf("first register: %v", err)
	}

	_, err := svc.Execute(context.Background(), cmd)
	if httpx.As(err).Code != "already_exists" {
		t.Fatalf("second register err = %v, want already_exists", err)
	}
}

// TestRegister_RollbackOnProfileFailure proves that a failure after the
// user insert rolls back the whole transaction (no orphan identity_users).
func TestRegister_RollbackOnProfileFailure(t *testing.T) {
	db := newDB(t)

	// Rebuild the register service with a failing patient API injected.
	hasher := auth.NewPasswordHasher()
	issuer := auth.NewTokenIssuer("test-secret-32-bytes-minimum-ok", 15*time.Minute)
	refresh := auth.NewRefreshStore(db, 24*time.Hour)
	svc := register.NewService(db, hasher, issuer, refresh,
		failingPatientsAPI{}, api.New(db, createsession.NewService(db)), &stubVerifier{})

	_, err := svc.Execute(context.Background(), register.Command{
		Email:       "rollback@example.com",
		Password:    "correct-horse-battery-staple",
		Role:        "Patient",
		FullName:    "Rollback User",
		PhoneNumber: "+201234567890",
	})
	if err == nil {
		t.Fatal("expected an error from the failing patient API")
	}

	var count int
	err = db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM identity_users WHERE email = $1`,
		"rollback@example.com").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("identity_users survived a rolled-back register (%d rows)", count)
	}
}

// failingPatientsAPI 500s the profile insert so the register transaction
// must roll back the already-inserted user row.
type failingPatientsAPI struct{}

func (failingPatientsAPI) CreatePatientProfile(
	ctx context.Context, in patientcontracts.CreatePatientProfileInput,
) (int64, error) {
	return 0, httpx.Internal(errors.New("injected patient profile failure"))
}

var _ patientcontracts.API = failingPatientsAPI{}
