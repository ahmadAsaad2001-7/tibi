//go:build integration

// Package integration holds the slice-18 integration test suite: real
// Postgres (via testcontainers), real migrations, real transactions.
// One container is started in TestMain and shared by every test in this
// package — a single `go test ./integration -tags=integration` run boots
// one Postgres, applies the schema once, and runs all suites against it.
package integration

import (
	"context"
	"testing"
	"time"

	"tibi/internal/doctors/api"
	"tibi/internal/doctors/createsession"
	"tibi/internal/identity/login"
	"tibi/internal/identity/register"
	patientsapi "tibi/internal/patients/createprofile"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/database"
	"tibi/internal/testutil"
)

func TestMain(m *testing.M) {
	testutil.RunMain(m)
}

// newDB returns a *database.DB over the shared pool with all tables reset.
func newDB(t *testing.T) *database.DB {
	t.Helper()
	pool := testutil.Postgres(t)
	testutil.TruncateAll(t, pool)
	return database.WrapPool(pool)
}

func newRegisterService(db *database.DB) *register.Service {
	hasher := auth.NewPasswordHasher()
	issuer := auth.NewTokenIssuer("test-secret-32-bytes-minimum-ok", 15*time.Minute)
	refresh := auth.NewRefreshStore(db, 24*time.Hour)
	return register.NewService(db, hasher, issuer, refresh,
		patientsapi.NewService(db), api.New(db, createsession.NewService(db)),
		&stubVerifier{})
}

func newLoginService(db *database.DB) *login.Service {
	hasher := auth.NewPasswordHasher()
	issuer := auth.NewTokenIssuer("test-secret-32-bytes-minimum-ok", 15*time.Minute)
	refresh := auth.NewRefreshStore(db, 24*time.Hour)
	return login.NewService(db, hasher, issuer, refresh)
}

type stubVerifier struct{}

func (stubVerifier) RequestVerification(ctx context.Context, userID int64) error {
	return nil
}
