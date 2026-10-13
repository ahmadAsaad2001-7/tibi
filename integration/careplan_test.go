//go:build integration

package integration

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"tibi/internal/clinical/careplan/cancelprotocol"
	"tibi/internal/clinical/careplan/createprotocol"
	"tibi/internal/clinical/careplan/listdoctorpatients"
	cpdb "tibi/internal/clinical/careplan/listdoctorpatients/db"
	"tibi/internal/clinical/careplan/updateprotocol"
	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/identity/register"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/testutil"
)

// Care-plan services are fully registered in cmd/api/main.go (HTTP) with BFF
// proxies under tibifront/src/app/api/care-plan and UI in
// care-plan-patient.tsx / care-plan-doctor.tsx. This suite drives the
// services directly against real Postgres — same harness as the auth suite.

// stubConsultations satisfies consultationscontracts.API. Care-plan tests never
// pass a consultation_id, so ClinicalContext must not be reached.
type stubConsultations struct{}

func (stubConsultations) Confirm(context.Context, int64) error {
	return errors.New("stub: not implemented")
}
func (stubConsultations) GetForCheckIn(context.Context, int64, int64) (*consultationscontracts.CheckInInfo, error) {
	return nil, errors.New("stub: not implemented")
}
func (stubConsultations) MarkCompleted(context.Context, int64) error {
	return errors.New("stub: not implemented")
}
func (stubConsultations) ClinicalContext(context.Context, int64, int64) (*consultationscontracts.ClinicalContextInfo, error) {
	return nil, errors.New("stub: not implemented")
}

func newCarePlanDB(t *testing.T) (*database.DB, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Postgres(t)
	testutil.TruncateAll(t, pool)
	return database.WrapPool(pool), pool
}

func registerCarePlanUser(t *testing.T, db *database.DB, email, role, phone string) int64 {
	t.Helper()
	resp, err := newRegisterService(db).Execute(context.Background(), register.Command{
		Email: email, Password: "correct-horse-battery-staple", Role: role,
		FullName: "Care Plan " + role, PhoneNumber: phone,
	})
	if err != nil {
		t.Fatalf("register %s: %v", role, err)
	}
	return resp.User.ID
}

func profileID(t *testing.T, pool *pgxpool.Pool, table string, userID int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		"SELECT id FROM "+table+" WHERE user_id = $1", userID).Scan(&id); err != nil {
		t.Fatalf("profile id from %s: %v", table, err)
	}
	return id
}

func verifyDoctor(t *testing.T, pool *pgxpool.Pool, userID int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"UPDATE doctors_profiles SET verification_status = 'Verified' WHERE user_id = $1", userID); err != nil {
		t.Fatalf("verify doctor: %v", err)
	}
}

func insertCompletion(t *testing.T, pool *pgxpool.Pool, protocolID int64, day time.Time, markedByUser int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		"INSERT INTO care_plan_completions (protocol_id, scheduled_date, marked_by_user_id) VALUES ($1, $2::date, $3)",
		protocolID, day.Format("2006-01-02"), markedByUser); err != nil {
		t.Fatalf("insert completion: %v", err)
	}
}

// calDate truncates t to its calendar date in t's own location, mirroring the
// services' dateOnly(time.Now()) so test and service agree on "today".
func calDate(tn time.Time) time.Time {
	y, m, d := tn.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, tn.Location())
}

func wantAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var e *httpx.APIError
	if !errors.As(err, &e) {
		t.Fatalf("want *httpx.APIError{status:%d code:%s}, got %v", status, code, err)
	}
	if e.Status != status || e.Code != code {
		t.Fatalf("want status=%d code=%q, got status=%d code=%q (msg %q)", status, code, e.Status, e.Code, e.Message)
	}
}

func sp(s string) *string { return &s }

// TestCarePlan_CreateValidatesGateAndEchoes covers the createprotocol contract:
// only Verified doctors may create, date/enum/reminder inputs are validated at
// the service boundary, unknown patients 404, and a valid command echoes back
// the normalized representation.
func TestCarePlan_CreateValidatesGateAndEchoes(t *testing.T) {
	db, pool := newCarePlanDB(t)
	doctorUser := registerCarePlanUser(t, db, "cp-doctor@example.com", "Doctor", "+201200000011")
	patientUser := registerCarePlanUser(t, db, "cp-patient@example.com", "Patient", "+201200000012")
	patientProfileID := profileID(t, pool, "patients_profiles", patientUser)
	svc := createprotocol.NewService(db, stubConsultations{})

	today := calDate(time.Now())
	start := today.Format("2006-01-02")
	end := today.AddDate(0, 0, 6).Format("2006-01-02")
	base := createprotocol.Command{
		PatientProfileID: patientProfileID, Title: "Metformin 500mg",
		Category: "Medication", Frequency: "Daily",
		StartDate: start, EndDate: &end,
	}

	// Gate first: a fresh doctor registers with verification_status=NotSubmitted.
	if _, err := svc.Execute(context.Background(), doctorUser, base); err == nil {
		t.Fatal("unverified doctor must not create a care plan")
	} else {
		wantAPIError(t, err, 403, "forbidden")
	}
	verifyDoctor(t, pool, doctorUser)

	// end_date < start_date.
	c := base
	c.EndDate = sp(today.AddDate(0, 0, -1).Format("2006-01-02"))
	wantAPIError(t, mustCreate(t, svc, doctorUser, c), 400, "validation_failed")

	// Malformed start_date.
	c = base
	c.StartDate = "04/10/2026"
	wantAPIError(t, mustCreate(t, svc, doctorUser, c), 400, "validation_failed")

	// Unknown frequency / category.
	c = base
	c.Frequency = "Sometimes"
	wantAPIError(t, mustCreate(t, svc, doctorUser, c), 400, "validation_failed")
	c = base
	c.Category = "Surgery"
	wantAPIError(t, mustCreate(t, svc, doctorUser, c), 400, "validation_failed")

	// Malformed reminder_time (must be HH:MM).
	c = base
	c.ReminderTime = sp("8am")
	wantAPIError(t, mustCreate(t, svc, doctorUser, c), 400, "validation_failed")

	// Patient that does not exist.
	c = base
	c.PatientProfileID = 999999999
	wantAPIError(t, mustCreate(t, svc, doctorUser, c), 404, "not_found")

	// Valid command: response echoes normalized fields.
	c = base
	c.ReminderTime = sp("08:00")
	resp, err := svc.Execute(context.Background(), doctorUser, c)
	if err != nil {
		t.Fatalf("valid create: %v", err)
	}
	docProfileID := profileID(t, pool, "doctors_profiles", doctorUser)
	if resp.ID <= 0 || resp.DoctorProfileID != docProfileID || resp.PatientProfileID != patientProfileID {
		t.Fatalf("ids: resp=%+v doctor=%d patient=%d", resp, docProfileID, patientProfileID)
	}
	if resp.StartDate != start || resp.EndDate == nil || *resp.EndDate != end {
		t.Fatalf("dates: start=%q end=%v want %q/%q", resp.StartDate, resp.EndDate, start, end)
	}
	if resp.ReminderTime == nil || *resp.ReminderTime != "08:00" {
		t.Fatalf("reminder_time = %v", resp.ReminderTime)
	}
	if resp.Category != "Medication" || resp.Frequency != "Daily" {
		t.Fatalf("category/frequency = %q/%q", resp.Category, resp.Frequency)
	}
}

func mustCreate(t *testing.T, svc *createprotocol.Service, userID int64, cmd createprotocol.Command) error {
	t.Helper()
	_, err := svc.Execute(context.Background(), userID, cmd)
	return err
}

// TestCarePlan_AdherenceFreezeCancel exercises the doctor dashboard contract:
// windowed 7d adherence math, today_completed/today_total, the reminder_time
// NULLS LAST + id ordering, the edit freeze after any completion, and cancel
// semantics (double-cancel 409, cancelled protocols vanish from the board).
func TestCarePlan_AdherenceFreezeCancel(t *testing.T) {
	db, pool := newCarePlanDB(t)
	doctorUser := registerCarePlanUser(t, db, "cp2-doctor@example.com", "Doctor", "+201200000021")
	patientUser := registerCarePlanUser(t, db, "cp2-patient@example.com", "Patient", "+201200000022")
	verifyDoctor(t, pool, doctorUser)
	patientProfileID := profileID(t, pool, "patients_profiles", patientUser)
	create := createprotocol.NewService(db, stubConsultations{})
	update := updateprotocol.NewService(db)
	cancel := cancelprotocol.NewService(db)
	list := listdoctorpatients.NewService(db)

	today := calDate(time.Now())
	startStr := today.AddDate(0, 0, -6).Format("2006-01-02")
	todayStr := today.Format("2006-01-02")
	endStr := todayStr // inclusive end => window covers today

	mk := func(title, freq, start, end, reminder string) int64 {
		t.Helper()
		cmd := createprotocol.Command{
			PatientProfileID: patientProfileID, Title: title,
			Category: "Other", Frequency: freq, StartDate: start,
		}
		if end != "" {
			cmd.EndDate = &end
		}
		if reminder != "" {
			cmd.ReminderTime = &reminder
		}
		resp, err := create.Execute(context.Background(), doctorUser, cmd)
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		return resp.ID
	}

	a := mk("A daily", "Daily", startStr, endStr, "08:00")       // expected 7d
	b := mk("B once", "Once", todayStr, "", "")                  // expected 1d, reminder NULL
	c := mk("C eod", "EveryOtherDay", startStr, endStr, "07:00") // expected 4d (offsets 0,2,4,6)
	dd := mk("D once", "Once", todayStr, "", "07:00")            // expected 1d, ties C on 07:00

	// Completions: A twice (today-2, today), C today. B/D incomplete.
	insertCompletion(t, pool, a, today.AddDate(0, 0, -2), patientUser)
	insertCompletion(t, pool, a, today, patientUser)
	insertCompletion(t, pool, c, today, patientUser)

	assertBoard := func(stage string, wantActive, wantDoneToday, wantTotal int, wantAdherence float64) {
		t.Helper()
		resp, err := list.Execute(context.Background(), doctorUser)
		if err != nil {
			t.Fatalf("[%s] list: %v", stage, err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("[%s] items = %d, want 1", stage, len(resp.Items))
		}
		row := resp.Items[0]
		if row.PatientProfileID != patientProfileID || row.ActiveProtocols != wantActive ||
			row.TodayCompleted != wantDoneToday || row.TodayTotal != wantTotal {
			t.Fatalf("[%s] row = %+v, want active=%d done=%d total=%d",
				stage, row, wantActive, wantDoneToday, wantTotal)
		}
		if math.Abs(row.Adherence7d-wantAdherence) > 1e-9 {
			t.Fatalf("[%s] adherence_7d = %v, want %v", stage, row.Adherence7d, wantAdherence)
		}
	}
	orderIDs := func(stage string, want []int64) {
		t.Helper()
		q := cpdb.New(pool)
		rows, err := q.ProtocolsActiveOn(context.Background(), cpdb.ProtocolsActiveOnParams{
			PatientProfileID: patientProfileID,
			ScheduledDate:    pgtypeDate(today),
		})
		if err != nil {
			t.Fatalf("[%s] ProtocolsActiveOn: %v", stage, err)
		}
		got := make([]int64, len(rows))
		for i, r := range rows {
			got[i] = r.ID
		}
		if len(got) != len(want) {
			t.Fatalf("[%s] order = %v, want %v", stage, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("[%s] order = %v, want %v (07:00 C,D by id, then 08:00 A, NULL B last)", stage, got, want)
			}
		}
	}

	// Expected today: A=7d, C=4d, B=1d, D=1d => 13. Done in window: 3.
	// Today: all four active, two completed (A, C).
	assertBoard("baseline", 4, 2, 4, 3.0/13.0)
	// Ordering: 07:00 group first (C then D by id), 08:00 (A), NULL reminder last (B).
	orderIDs("baseline", []int64{c, dd, a, b})

	// Any completion freezes edits on A, but not on completion-free D.
	if _, err := update.Execute(context.Background(), doctorUser, a, updateprotocol.Command{Title: sp("edited")}); err == nil {
		t.Fatal("frozen protocol must reject updates")
	} else {
		wantAPIError(t, err, 409, "protocol_frozen")
	}
	newTitle := "D once edited"
	if _, err := update.Execute(context.Background(), doctorUser, dd, updateprotocol.Command{Title: &newTitle}); err != nil {
		t.Fatalf("update unfrozen protocol: %v (cause: %v)", err, httpx.As(err).Err)
	}

	// Patients cannot cancel: cancel resolves the caller's doctor profile first.
	if _, err := cancel.Execute(context.Background(), patientUser, a); err == nil {
		t.Fatal("patient must not cancel a protocol")
	} else {
		wantAPIError(t, err, 403, "forbidden")
	}

	// Cancel A: excluded from board, adherence drops to C(1/4)+B+D = 1/6.
	if _, err := cancel.Execute(context.Background(), doctorUser, a); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := cancel.Execute(context.Background(), doctorUser, a); err == nil {
		t.Fatal("double cancel must fail")
	} else {
		wantAPIError(t, err, 409, "already_cancelled")
	}
	if _, err := update.Execute(context.Background(), doctorUser, a, updateprotocol.Command{Title: sp("late edit")}); err == nil {
		t.Fatal("cancelled protocol must reject updates")
	} else {
		wantAPIError(t, err, 409, "already_cancelled")
	}
	assertBoard("after cancel", 3, 1, 3, 1.0/6.0)
	orderIDs("after cancel", []int64{c, dd, b})
}

// pgtypeDate builds the pgtype.Date used by the sqlc queries for "today".
func pgtypeDate(day time.Time) pgtype.Date {
	return pgtype.Date{Time: day, Valid: true}
}
