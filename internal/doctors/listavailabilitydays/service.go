package listavailabilitydays

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/availability"
	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/listavailabilitydays/db"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/weeklyschedule"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

const maxRangeDays = 60

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type DayOut struct {
	Date      string `json:"date"`
	SlotCount int    `json:"slot_count"`
	FirstSlot string `json:"first_slot"`
}

type Response struct {
	Days []DayOut `json:"days"`
}

func (s *Service) Execute(ctx context.Context, doctorProfileID int64, from, to time.Time) (*Response, error) {
	if to.Before(from) {
		return nil, httpx.ValidationFailed(map[string]string{"to": "must be after from"})
	}
	if to.Sub(from) > maxRangeDays*24*time.Hour {
		return nil, httpx.ValidationFailed(map[string]string{"to": "range exceeds 60 days"})
	}

	q := db.New(s.db.Querier(ctx))

	if _, err := q.GetVerifiedDoctor(ctx, doctorProfileID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor not found")
		}
		return nil, httpx.Internal(err)
	}

	blockRows, err := q.BlocksInRange(ctx, doctorProfileID)
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

	exceptionRows, err := q.ExceptionsInRange(ctx, db.ExceptionsInRangeParams{
		DoctorProfileID: doctorProfileID,
		ExceptionDate:   pgtypeDate(from),
		ExceptionDate_2: pgtypeDate(to),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	exceptionsByDate := map[string][]scheduleexception.Exception{}
	for _, e := range exceptionRows {
		key := e.ExceptionDate.Time.Format("2006-01-02")
		exceptionsByDate[key] = append(exceptionsByDate[key], scheduleexception.Exception{
			Date:     e.ExceptionDate.Time,
			FromTime: timeFromPg(e.FromTime),
			ToTime:   timeFromPg(e.ToTime),
			Type:     scheduleexception.Type(e.Type),
		})
	}

	bookedRows, err := q.BookedInRange(ctx, doctorProfileID, from, to.AddDate(0, 0, 1))
	if err != nil {
		return nil, httpx.Internal(err)
	}
	bookedByDate := map[string][]doctorprofile.TimeOfDay{}
	for _, at := range bookedRows {
		u := at.UTC()
		key := u.Format("2006-01-02")
		bookedByDate[key] = append(bookedByDate[key], doctorprofile.TimeOfDay{Hour: u.Hour(), Minute: u.Minute()})
	}

	resp := &Response{Days: []DayOut{}}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		slots := availability.ForDate(d, blocks, exceptionsByDate[key], bookedByDate[key])
		if len(slots) == 0 {
			continue
		}
		resp.Days = append(resp.Days, DayOut{
			Date:      key,
			SlotCount: len(slots),
			FirstSlot: slots[0].Start.String(),
		})
	}
	return resp, nil
}

func timeFromPg(t pgtype.Time) doctorprofile.TimeOfDay {
	total := int(t.Microseconds / 1_000_000)
	return doctorprofile.TimeOfDay{Hour: total / 3600, Minute: (total % 3600) / 60}
}

func pgtypeDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}
