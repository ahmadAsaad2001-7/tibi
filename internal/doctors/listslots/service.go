package listslots

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/availability"
	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/listslots/db"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/weeklyschedule"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type SlotOut struct {
	Start           string `json:"start"`
	DurationMinutes int    `json:"duration_minutes"`
}

type Response struct {
	Date  string    `json:"date"`
	Slots []SlotOut `json:"slots"`
}

func (s *Service) Execute(ctx context.Context, doctorProfileID int64, date time.Time) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	if _, err := q.GetVerifiedDoctor(ctx, doctorProfileID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor not found")
		}
		return nil, httpx.Internal(err)
	}

	blockRows, err := q.BlocksForDoctor(ctx, doctorProfileID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	blocks := make([]weeklyschedule.Block, len(blockRows))
	for i, b := range blockRows {
		blocks[i] = weeklyschedule.Block{
			DayOfWeek:           int(b.DayOfWeek),
			StartTime:           timeFromPg(b.StartTime),
			EndTime:             timeFromPg(b.EndTime),
			SlotDurationMinutes: int(b.SlotDurationMinutes),
			IsActive:            b.IsActive,
		}
	}

	// ✅ FIX 1: استخدام ExceptionsOnDateParams
	exceptionRows, err := q.ExceptionsOnDate(ctx, db.ExceptionsOnDateParams{
		DoctorProfileID: doctorProfileID,
		ExceptionDate:   pgtype.Date{Time: date, Valid: true},
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	exceptions := make([]scheduleexception.Exception, len(exceptionRows))
	for i, e := range exceptionRows {
		exceptions[i] = scheduleexception.Exception{
			Date:     e.ExceptionDate.Time, // ✅ FIX 2: استخراج .Time من pgtype.Date
			FromTime: timeFromPg(e.FromTime),
			ToTime:   timeFromPg(e.ToTime),
			Type:     scheduleexception.Type(e.Type),
		}
	}

	// ✅ FIX 3: استخدام BookedOnDateParams
	bookedRows, err := q.BookedOnDate(ctx, db.BookedOnDateParams{
		DoctorProfileID: doctorProfileID,
		FromTs:          pgtype.Timestamptz{Time: date, Valid: true},
		ToTs:            pgtype.Timestamptz{Time: date.AddDate(0, 0, 1), Valid: true},
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	booked := make([]doctorprofile.TimeOfDay, len(bookedRows))
	for i, at := range bookedRows {
		// ✅ FIX 4: at هو pgtype.Timestamptz، نحتاج لاستخراج .Time قبل استدعاء UTC()
		u := at.Time.UTC()
		booked[i] = doctorprofile.TimeOfDay{
			Hour:   u.Hour(),
			Minute: u.Minute(),
		}
	}

	slots := availability.ForDate(date, blocks, exceptions, booked)
	out := make([]SlotOut, len(slots))
	for i, slot := range slots {
		out[i] = SlotOut{Start: slot.Start.String(), DurationMinutes: int(slot.Duration / time.Minute)}
	}

	return &Response{Date: date.Format("2006-01-02"), Slots: out}, nil
}

func timeFromPg(t pgtype.Time) doctorprofile.TimeOfDay {
	total := int(t.Microseconds / 1_000_000)
	return doctorprofile.TimeOfDay{Hour: total / 3600, Minute: (total % 3600) / 60}
}
