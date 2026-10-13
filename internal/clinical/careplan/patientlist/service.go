package patientlist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan/patientlist/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

type ProtocolRow struct {
	ID               int64   `json:"id"`
	PatientProfileID int64   `json:"patient_profile_id"`
	DoctorProfileID  int64   `json:"doctor_profile_id"`
	Title            string  `json:"title"`
	Description      *string `json:"description,omitempty"`
	Category         string  `json:"category"`
	Frequency        string  `json:"frequency"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date,omitempty"`
	ReminderTime     *string `json:"reminder_time,omitempty"`
	IsCompletedToday bool    `json:"is_completed_today"`
	CompletionAt     *string `json:"completion_at,omitempty"`
	CompletionNote   *string `json:"completion_note,omitempty"`
}

type Response struct {
	Items []ProtocolRow `json:"items"`
}

func (s *Service) Execute(ctx context.Context, userID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))

	profileID, err := q.GetPatientProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("patient profile not found")
		}
		return nil, httpx.Internal(err)
	}

	today := pgtype.Date{Time: time.Now(), Valid: true}
	rows, err := q.PatientProtocols(ctx, profileID, today)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	items := make([]ProtocolRow, 0, len(rows))
	for i := range rows {
		row := rows[i]
		protocol := ProtocolRow{
			ID:               row.ID,
			PatientProfileID: row.PatientProfileID,
			DoctorProfileID:  row.DoctorProfileID,
			Title:            row.Title,
			Description:      row.Description,
			Category:         string(row.Category),
			Frequency:        string(row.Frequency),
			StartDate:        row.StartDate.Time.Format("2006-01-02"),
			EndDate:          datePtr(row.EndDate),
			ReminderTime:     timePtr(row.ReminderTime),
			IsCompletedToday: row.CompletionMarkedAt.Valid,
		}
		if row.CompletionMarkedAt.Valid {
			protocol.CompletionAt = ptrTime(row.CompletionMarkedAt.Time)
			protocol.CompletionNote = row.CompletionNote
		}
		items = append(items, protocol)
	}

	return &Response{Items: items}, nil
}

func datePtr(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func timePtr(t pgtype.Time) *string {
	if !t.Valid {
		return nil
	}
	secs := t.Microseconds / 1_000_000
	s := fmt.Sprintf("%02d:%02d", secs/3600, (secs%3600)/60)
	return &s
}

func ptrTime(t time.Time) *string {
	s := t.Format(time.RFC3339)
	return &s
}
