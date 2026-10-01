package clinicsession

import (
	"time"

	"tibi/internal/doctors/doctorprofile"
)

// Session is one working block on one date. Materializes a weekly schedule
// block for a specific day. Created lazily on first booking.
type Session struct {
	ID              int64
	DoctorProfileID int64
	Date            time.Time
	StartTime       doctorprofile.TimeOfDay
	EndTime         doctorprofile.TimeOfDay
	CreatedAt       time.Time
}
