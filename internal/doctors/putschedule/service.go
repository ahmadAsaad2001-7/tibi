package putschedule

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/putschedule/db"
	"tibi/internal/doctors/weeklyschedule"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	Weekly []BlockOut `json:"weekly"`
}

type BlockOut struct {
	DayOfWeek           int    `json:"day_of_week"`
	StartTime           string `json:"start_time"`
	EndTime             string `json:"end_time"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
	IsActive            bool   `json:"is_active"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	// Parse and validate outside the transaction. Fail fast.
	blocks := make([]weeklyschedule.Block, len(cmd.Weekly))
	for i, b := range cmd.Weekly {
		start, err := doctorprofile.ParseTimeOfDay(b.StartTime)
		if err != nil {
			return nil, httpx.ValidationFailed(map[string]string{
				"weekly": "block " + itoa(i) + ": invalid start_time",
			})
		}
		end, err := doctorprofile.ParseTimeOfDay(b.EndTime)
		if err != nil {
			return nil, httpx.ValidationFailed(map[string]string{
				"weekly": "block " + itoa(i) + ": invalid end_time",
			})
		}
		blocks[i] = weeklyschedule.Block{
			DayOfWeek:           b.DayOfWeek,
			StartTime:           start,
			EndTime:             end,
			SlotDurationMinutes: b.SlotDurationMinutes,
			IsActive:            b.IsActive,
		}
	}

	var resp *Response
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		profileID, err := q.GetDoctorProfileIDForSchedule(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("doctor profile not found")
			}
			return httpx.Internal(err)
		}

		sched := weeklyschedule.New(profileID, blocks)
		if err := sched.Validate(); err != nil {
			return httpx.ValidationFailed(map[string]string{"weekly": err.Error()})
		}

		if err := q.DeleteAllBlocks(ctx, profileID); err != nil {
			return httpx.Internal(err)
		}
		for _, b := range blocks {
			if err := q.InsertBlock(ctx, db.InsertBlockParams{
				DoctorProfileID:     profileID,
				DayOfWeek:           int16(b.DayOfWeek),
				StartTime:           pgtypeTime(b.StartTime),
				EndTime:             pgtypeTime(b.EndTime),
				SlotDurationMinutes: int32(b.SlotDurationMinutes),
				IsActive:            b.IsActive,
			}); err != nil {
				return httpx.Internal(err)
			}
		}

		resp = &Response{Weekly: make([]BlockOut, len(blocks))}
		for i, b := range blocks {
			resp.Weekly[i] = BlockOut{
				DayOfWeek:           b.DayOfWeek,
				StartTime:           b.StartTime.String(),
				EndTime:             b.EndTime.String(),
				SlotDurationMinutes: b.SlotDurationMinutes,
				IsActive:            b.IsActive,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func itoa(i int) string { return strconv.Itoa(i) }

func pgtypeTime(t doctorprofile.TimeOfDay) pgtype.Time {
	return pgtype.Time{
		Microseconds: int64(t.Hour)*3600_000_000 + int64(t.Minute)*60_000_000,
		Valid:        true,
	}
}
