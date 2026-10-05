package search

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/search/db"
	"tibi/internal/doctors/weeklyschedule"
)

func ptr[T any](v T) *T { return &v }

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func cursorFromRow(sort Sort, r searchRow) cursor {
	c := cursor{ID: r.ID}
	switch sort {
	case SortRating:
		c.Rating = &r.AverageRating
	case SortFeeAsc, SortFeeDesc:
		c.Fee = &r.ConsultationFee
	case SortCreatedAt:
		// ✅ FIX: استخراج .Time من pgtype.Timestamptz قبل استدعاء .UTC()
		s := r.CreatedAt.Time.UTC().Format(time.RFC3339Nano)
		c.CreatedAt = &s
	}
	return c
}

// Updated to match exact sqlc generated type names
func groupSpecialties(rows []db.SpecialtiesForDoctorsRow) map[int64][]Specialty {
	out := map[int64][]Specialty{}
	for _, r := range rows {
		out[r.DoctorProfileID] = append(out[r.DoctorProfileID], Specialty{ID: r.SpecialtyID, Name: r.Name})
	}
	return out
}

func groupBlocks(rows []db.BlocksForDoctorsRow) map[int64][]weeklyschedule.Block {
	out := map[int64][]weeklyschedule.Block{}
	for _, r := range rows {
		out[r.DoctorProfileID] = append(out[r.DoctorProfileID], weeklyschedule.Block{
			DayOfWeek:           int(r.DayOfWeek),
			StartTime:           tod(r.StartTime),
			EndTime:             tod(r.EndTime),
			SlotDurationMinutes: int(r.SlotDurationMinutes),
			IsActive:            r.IsActive,
		})
	}
	return out
}

func groupExceptions(rows []db.ExceptionsForDoctorsRow) map[int64][]scheduleexception.Exception {
	out := map[int64][]scheduleexception.Exception{}
	for _, r := range rows {
		out[r.DoctorProfileID] = append(out[r.DoctorProfileID], scheduleexception.Exception{
			DoctorProfileID: r.DoctorProfileID,
			Date:            r.ExceptionDate.Time, // pgtype.Date uses .Time
			FromTime:        tod(r.FromTime),
			ToTime:          tod(r.ToTime),
			Type:            scheduleexception.Type(r.Type),
		})
	}
	return out
}

func groupBooked(rows []db.BookedSlotsForDoctorsRow) map[int64]map[string][]doctorprofile.TimeOfDay {
	out := map[int64]map[string][]doctorprofile.TimeOfDay{}
	for _, r := range rows {
		u := r.ScheduledAt.Time.UTC() // pgtype.Timestamptz uses .Time
		day := u.Format("2006-01-02")
		m := out[r.DoctorProfileID]
		if m == nil {
			m = map[string][]doctorprofile.TimeOfDay{}
			out[r.DoctorProfileID] = m
		}
		m[day] = append(m[day], doctorprofile.TimeOfDay{Hour: u.Hour(), Minute: u.Minute()})
	}
	return out
}

func tod(t pgtype.Time) doctorprofile.TimeOfDay {
	secs := int(t.Microseconds / 1_000_000)
	return doctorprofile.TimeOfDay{Hour: secs / 3600, Minute: (secs % 3600) / 60}
}
