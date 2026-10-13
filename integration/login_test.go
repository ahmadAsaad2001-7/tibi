//go:build integration

package integration

import (
	"context"
	"testing"

	"tibi/internal/identity/login"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

func seedUser(t *testing.T, db *database.DB, email, password string) int64 {
	t.Helper()
	hasher := auth.NewPasswordHasher()
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}

	var id int64
	err = db.Pool.QueryRow(context.Background(),
		`INSERT INTO identity_users (email, password_hash, role)
		 VALUES ($1, $2, 'Patient') RETURNING id`,
		email, hash).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestLogin_WithMatchingPassword(t *testing.T) {
	db := newDB(t)
	seedUser(t, db, "user@example.com", "correct-horse-battery-staple")
	svc := newLoginService(db)

	resp, err := svc.Execute(context.Background(), login.Command{
		Email:    "user@example.com",
		Password: "correct-horse-battery-staple",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}
}

// TestLogin_WrongPassword returns the identical error to a missing user —
// no user enumeration.
func TestLogin_WrongPassword(t *testing.T) {
	db := newDB(t)
	seedUser(t, db, "user@example.com", "correct-horse-battery-staple")
	svc := newLoginService(db)

	_, err := svc.Execute(context.Background(), login.Command{
		Email:    "user@example.com",
		Password: "wrong-password",
	})
	if httpx.As(err).Code != "unauthenticated" {
		t.Fatalf("wrong password err = %v, want unauthenticated", err)
	}
}

func TestLogin_MissingUser(t *testing.T) {
	db := newDB(t)
	svc := newLoginService(db)

	_, err := svc.Execute(context.Background(), login.Command{
		Email:    "ghost@example.com",
		Password: "whatever-password",
	})
	if httpx.As(err).Code != "unauthenticated" {
		t.Fatalf("missing user err = %v, want unauthenticated", err)
	}
}
