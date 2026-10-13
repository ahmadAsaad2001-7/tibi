//go:build integration

package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TruncateAll empties every application table. Called from a test's setup
// to start from a known state. Schema is preserved.
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	// Order-independent because CASCADE. RESTART IDENTITY resets the
	// auto-increment counters so tests can assume id=1 for the first insert.
	_, err := pool.Exec(ctx, `
		TRUNCATE
			identity_password_resets,
			identity_email_verifications,
			identity_refresh_tokens,
			platform_rate_limits,
			communication_notifications,
			communication_messages,
			doctorposts_free_messages,
			content_post_attachments,
			content_posts,
			clinical_prescribed_medications,
			clinical_prescriptions,
			clinical_medical_attachments,
			clinical_medical_records,
			admin_vote_participants,
			admin_votes,
			admin_user_suspensions,
			queue_entries,
			queue_windows,
			payments_payments,
			consultations_consultations,
			doctors_clinic_sessions,
			doctors_profile_specialties,
			doctors_schedule_exceptions,
			doctors_weekly_schedules,
			doctors_profiles,
			doctors_specialties,
			patients_profiles,
			identity_users,
			platform_files
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
