package getschedule

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/getschedule/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type BlockOut struct {
	DayOfWeek           int    `json:"day_of_week"`
	StartTime           string `json:"start_time"`
	EndTime             string `json:"end_time"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
	IsActive            bool   `json:"is_active"`
}

type ExceptionOut struct {
	ID       int64   `json:"id"`
	Date     string  `json:"date"`
	FromTime string  `json:"from_time"`
	ToTime   string  `json:"to_time"`
	Type     string  `json:"type"`
	Reason   *string `json:"reason"`
}

type Response struct {
	Weekly     []BlockOut     `json:"weekly"`
	Exceptions []ExceptionOut `json:"exceptions"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	profileID, err := q.GetDoctorProfileID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}

	blockRows, err := q.ListBlocks(ctx, profileID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	exceptionRows, err := q.ListExceptions(ctx, profileID)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	weekly := make([]BlockOut, len(blockRows))
	for i, b := range blockRows {
		start, end := timeFromPg(b.StartTime), timeFromPg(b.EndTime)
		weekly[i] = BlockOut{
			DayOfWeek:           int(b.DayOfWeek),
			StartTime:           start.String(),
			EndTime:             end.String(),
			SlotDurationMinutes: int(b.SlotDurationMinutes),
			IsActive:            b.IsActive,
		}
	}
	exceptions := make([]ExceptionOut, len(exceptionRows))
	for i, e := range exceptionRows {
		exceptions[i] = ExceptionOut{
			ID:       e.ID,
			Date:     e.ExceptionDate.Time.Format("2006-01-02"),
			FromTime: timeFromPg(e.FromTime).String(),
			ToTime:   timeFromPg(e.ToTime).String(),
			Type:     e.Type,
			Reason:   e.Reason,
		}
	}
	return &Response{Weekly: weekly, Exceptions: exceptions}, nil
}

func timeFromPg(t pgtype.Time) doctorprofile.TimeOfDay {
	total := int(t.Microseconds / 1_000_000)
	return doctorprofile.TimeOfDay{Hour: total / 3600, Minute: (total % 3600) / 60}
}
