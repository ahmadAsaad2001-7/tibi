package seed

import (
	"context"
	"fmt"
	"os"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/database"
)

type Options struct {
	Production bool
}

// Run applies the minimum data required for a fresh deployment:
//   - Specialty catalog (idempotent: ON CONFLICT DO NOTHING)
//   - One admin user (only if ADMIN_EMAIL + ADMIN_PASSWORD are set)
//
// It never seeds doctors, patients, or any user-supplied data.
func Run(ctx context.Context, db *database.DB, opts Options) error {
	if err := seedSpecialties(ctx, db); err != nil {
		return fmt.Errorf("specialties: %w", err)
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminEmail != "" && adminPassword != "" {
		if err := seedAdmin(ctx, db, adminEmail, adminPassword); err != nil {
			return fmt.Errorf("admin: %w", err)
		}
	} else if opts.Production {
		fmt.Println("warning: ADMIN_EMAIL/ADMIN_PASSWORD not set; no admin seeded")
	}
	return nil
}

// seedSpecialties mirrors migration 0002 but with ON CONFLICT DO NOTHING,
// so running it after migrations is a no-op (never duplicates).
func seedSpecialties(ctx context.Context, db *database.DB) error {
	roots := []string{
		"General Practice",
		"Cardiology",
		"Dermatology",
		"Pediatrics",
		"Psychiatry",
		"Orthopedics",
		"Internal Medicine",
		"Neurology",
	}
	q := db.Querier(ctx)
	for _, name := range roots {
		if _, err := q.Exec(ctx,
			`INSERT INTO doctors_specialties (name, parent_specialty_id)
			 VALUES ($1, NULL)
			 ON CONFLICT (name) DO NOTHING`,
			name); err != nil {
			return err
		}
	}

	children := []struct {
		name   string
		parent string
	}{
		{"Interventional Cardiology", "Cardiology"},
		{"Pediatric Cardiology", "Cardiology"},
	}
	for _, ch := range children {
		if _, err := q.Exec(ctx,
			`INSERT INTO doctors_specialties (name, parent_specialty_id)
			 SELECT $1, id FROM doctors_specialties WHERE name = $2
			 ON CONFLICT (name) DO NOTHING`,
			ch.name, ch.parent); err != nil {
			return err
		}
	}
	return nil
}

// seedAdmin creates an Admin user if none exists with that email. The
// email address is unique in the schema, so a duplicate insert is safe to
// surface as an error rather than silently swallow.
func seedAdmin(ctx context.Context, db *database.DB, email, password string) error {
	hasher := auth.NewPasswordHasher()
	hash, err := hasher.Hash(password)
	if err != nil {
		return err
	}

	q := db.Querier(ctx)
	var exists bool
	err = q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM identity_users WHERE email = $1)`,
		email).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = q.Exec(ctx,
		`INSERT INTO identity_users (email, password_hash, role)
		 VALUES ($1, $2, 'Admin')`,
		email, hash)
	return err
}
