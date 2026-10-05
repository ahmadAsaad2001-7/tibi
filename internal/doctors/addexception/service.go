package addexception

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/addexception/db"
	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	ID       int64   `json:"id"`
	Date     string  `json:"date"`
	FromTime string  `json:"from_time"`
	ToTime   string  `json:"to_time"`
	Type     string  `json:"type"`
	Reason   *string `json:"reason"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	day, err := time.Parse("2006-01-02", cmd.Date)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"date": "expected YYYY-MM-DD"})
	}
	from, err := doctorprofile.ParseTimeOfDay(cmd.FromTime)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"from_time": "expected HH:MM"})
	}
	to, err := doctorprofile.ParseTimeOfDay(cmd.ToTime)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"to_time": "expected HH:MM"})
	}
	kind := scheduleexception.Type(cmd.Type)
	if !kind.Valid() {
		return nil, httpx.ValidationFailed(map[string]string{"type": "unknown exception type"})
	}
	if !from.Before(to) {
		return nil, httpx.ValidationFailed(map[string]string{"from_time": "must be before to_time"})
	}

	q := db.New(s.db.Querier(ctx))
	profileID, err := q.GetDoctorProfileID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}

	// ✅ FIX 5: تحويل *string إلى pgtype.Text بأمان
	reasonText := pgtype.Text{Valid: false}
	if cmd.Reason != nil {
		reasonText = pgtype.Text{String: *cmd.Reason, Valid: true}
	}

	id, err := q.InsertException(ctx, db.InsertExceptionParams{
		DoctorProfileID: profileID,
		ExceptionDate:   pgtype.Date{Time: day, Valid: true}, // ✅ FIX 1: تحويل time.Time إلى pgtype.Date
		FromTime:        pgtypeTime(from),                    // ✅ FIX 2: تحويل TimeOfDay إلى pgtype.Time
		ToTime:          pgtypeTime(to),                      // ✅ FIX 3: تحويل TimeOfDay إلى pgtype.Time
		Type:            db.ScheduleExceptionType(kind),      // ✅ FIX 4: تحويل scheduleexception.Type إلى db.ScheduleExceptionType
		Reason:          reasonText,                          // ✅ استخدام المتغير المحول
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	return &Response{
		ID:       id,
		Date:     day.Format("2006-01-02"),
		FromTime: from.String(),
		ToTime:   to.String(),
		Type:     string(kind),
		Reason:   cmd.Reason,
	}, nil
}

func pgtypeTime(t doctorprofile.TimeOfDay) pgtype.Time {
	return pgtype.Time{
		Microseconds: int64(t.Hour)*3_600_000_000 + int64(t.Minute)*60_000_000,
		Valid:        true,
	}
}
