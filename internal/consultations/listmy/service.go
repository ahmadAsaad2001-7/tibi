package listmy

import (
	"context"
	"time"

	"tibi/internal/consultations/listmy/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Item struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	ScheduledAt      time.Time `json:"scheduled_at"`
	DurationMinutes  int       `json:"duration_minutes"`
	IsUrgent         bool      `json:"is_urgent"`
	DoctorProfileID  int64     `json:"doctor_profile_id"`
	DoctorName       string    `json:"doctor_name"`
	PatientProfileID int64     `json:"patient_profile_id"`
	PatientName      string    `json:"patient_name"`
}

type Response struct {
	Items []Item `json:"items"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	rows, err := db.New(s.db.Querier(ctx)).ListForUser(ctx, userID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	items := make([]Item, len(rows))
	for i, r := range rows {
		items[i] = Item{
			ID:               r.ID,
			Status:           string(r.Status),
			ScheduledAt:      r.ScheduledAt.Time,
			DurationMinutes:  int(r.DurationMinutes),
			IsUrgent:         r.IsUrgent,
			DoctorProfileID:  r.DoctorProfileID,
			DoctorName:       r.DoctorName,
			PatientProfileID: r.PatientProfileID,
			PatientName:      r.PatientName,
		}
	}
	return &Response{Items: items}, nil
}
