package getbyid

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/consultations/getbyid/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Response struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	ScheduledAt      time.Time `json:"scheduled_at"`
	DurationMinutes  int       `json:"duration_minutes"`
	IsUrgent         bool      `json:"is_urgent"`
	Notes            *string   `json:"notes"`
	DoctorProfileID  int64     `json:"doctor_profile_id"`
	DoctorName       string    `json:"doctor_name"`
	PatientProfileID int64     `json:"patient_profile_id"`
	PatientName      string    `json:"patient_name"`
	CreatedAt        time.Time `json:"created_at"`
}

func (s *Service) Execute(ctx context.Context, userID, id int64) (*Response, error) {
	row, err := db.New(s.db.Querier(ctx)).GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("consultation not found")
		}
		return nil, httpx.Internal(err)
	}
	if userID != row.PatientUserID && userID != row.DoctorUserID {
		return nil, httpx.Forbidden("not a participant")
	}

	// ✅ FIX: تحويل pgtype.Text إلى *string بأمان
	var notes *string
	if row.Notes.Valid {
		notes = &row.Notes.String
	}

	return &Response{
		ID:               row.ID,
		Status:           string(row.Status),   // ✅ FIX 1: تحويل db.ConsultationStatus إلى string
		ScheduledAt:      row.ScheduledAt.Time, // ✅ FIX 2: استخراج time.Time من pgtype.Timestamptz
		DurationMinutes:  int(row.DurationMinutes),
		IsUrgent:         row.IsUrgent,
		Notes:            notes, // ✅ FIX 3: استخدام المتغير المحول
		DoctorProfileID:  row.DoctorProfileID,
		DoctorName:       row.DoctorName,
		PatientProfileID: row.PatientProfileID,
		PatientName:      row.PatientName,
		CreatedAt:        row.CreatedAt.Time, // ✅ FIX 4: استخراج time.Time من pgtype.Timestamptz
	}, nil
}
