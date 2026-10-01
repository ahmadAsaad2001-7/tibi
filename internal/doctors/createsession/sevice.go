package createsession

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctors/createsession/db"
	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

// ValidateSlot verifies the doctor is verified, the slot falls within an
// active weekly block, and no closed exception covers it. Returns the block
// boundaries so the caller can ensure the session.
type SlotInfo struct {
	BlockStartTime      doctorprofile.TimeOfDay
	BlockEndTime        doctorprofile.TimeOfDay
	SlotDurationMinutes int
}

func (s *Service) ValidateSlot(ctx context.Context, doctorProfileID int64, scheduledAt time.Time) (*SlotInfo, error) {
	q := db.New(s.db.Querier(ctx))

	doc, err := q.GetDoctorProfileByID(ctx, doctorProfileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor not found")
		}
		return nil, httpx.Internal(err)
	}
	if doc.VerificationStatus != "Verified" {
		return nil, httpx.Unprocessable("doctor is not verified")
	}

	slotStart := doctorprofile.TimeOfDay{Hour: scheduledAt.Hour(), Minute: scheduledAt.Minute()}
	slotEnd := addMinutes(slotStart, 1) // not used for block match; block match uses slot start only

	block, err := q.GetDoctorBlockForDate(ctx, db.GetDoctorBlockForDateParams{
		DoctorProfileID: doctorProfileID,
		Date:            pgDate(scheduledAt),
		StartTime:       pgTime(slotStart),
		EndTime:         pgTime(slotEnd),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Unprocessable("slot is outside the doctor's schedule")
		}
		return nil, httpx.Internal(err)
	}

	if _, err := q.GetClosedException(ctx, db.GetClosedExceptionParams{
		DoctorProfileID: doctorProfileID,
		ExceptionDate:   pgDate(scheduledAt),
		FromTime:        pgTime(slotStart),
		ToTime:          pgTime(slotEnd),
	}); err == nil {
		return nil, httpx.Unprocessable("doctor is closed on this date")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.Internal(err)
	}

	return &SlotInfo{
		BlockStartTime:      pgTimeToTOD(block.StartTime),
		BlockEndTime:        pgTimeToTOD(block.EndTime),
		SlotDurationMinutes: int(block.SlotDurationMinutes),
	}, nil
}

// EnsureSession inserts or returns the session for the block containing the
// slot. Called by the booking slice inside its transaction.
func (s *Service) EnsureSession(ctx context.Context, doctorProfileID int64, date time.Time, info *SlotInfo) (int64, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetOrCreateSession(ctx, db.GetOrCreateSessionParams{
		DoctorProfileID: doctorProfileID,
		SessionDate:     pgDate(date),
		StartTime:       pgTime(info.BlockStartTime),
		EndTime:         pgTime(info.BlockEndTime),
	})
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return row.ID, nil
}

// helper signatures stubbed; see slice-4 pgTime/pgDate
func addMinutes(t doctorprofile.TimeOfDay, m int) doctorprofile.TimeOfDay {
	total := t.Minutes() + m
	return doctorprofile.TimeOfDay{Hour: total / 60, Minute: total % 60}
}
