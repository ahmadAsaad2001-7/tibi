package createsession

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/doctorprofile"
)

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

func pgTime(t doctorprofile.TimeOfDay) pgtype.Time {
	return pgtype.Time{
		Microseconds: int64(t.Hour)*3_600_000_000 + int64(t.Minute)*60_000_000,
		Valid:        true,
	}
}

func pgTimeToTOD(t pgtype.Time) doctorprofile.TimeOfDay {
	secs := int(t.Microseconds / 1_000_000)
	return doctorprofile.TimeOfDay{Hour: secs / 3600, Minute: (secs % 3600) / 60}
}
