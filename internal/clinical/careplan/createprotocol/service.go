package createprotocol

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan"
	"tibi/internal/clinical/careplan/createprotocol/db"
	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db            *database.DB
	consultations consultationscontracts.API
	clock         func() time.Time
}

func NewService(db *database.DB, c consultationscontracts.API) *Service {
	return &Service{db: db, consultations: c, clock: time.Now}
}

type Command struct {
	PatientProfileID int64   `json:"patient_profile_id" validate:"required,gt=0"`
	ConsultationID   *int64  `json:"consultation_id" validate:"omitempty,gt=0"`
	Title            string  `json:"title" validate:"required,min=1,max=200"`
	Description      *string `json:"description" validate:"omitempty,max=1000"`
	Category         string  `json:"category" validate:"required,oneof=Medication Vitals Lifestyle FollowUp Other"`
	Frequency        string  `json:"frequency" validate:"required,oneof=Daily EveryOtherDay Weekly Once"`
	StartDate        string  `json:"start_date" validate:"required"`
	EndDate          *string `json:"end_date" validate:"omitempty"`
	ReminderTime     *string `json:"reminder_time" validate:"omitempty"`
}

type Response struct {
	ID               int64   `json:"id"`
	PatientProfileID int64   `json:"patient_profile_id"`
	DoctorProfileID  int64   `json:"doctor_profile_id"`
	ConsultationID   *int64  `json:"consultation_id"`
	Title            string  `json:"title"`
	Description      *string `json:"description"`
	Category         string  `json:"category"`
	Frequency        string  `json:"frequency"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date"`
	ReminderTime     *string `json:"reminder_time"`
	CreatedAt        string  `json:"created_at"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	start, err := time.Parse("2006-01-02", cmd.StartDate)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"start_date": "expected YYYY-MM-DD"})
	}
	var end *time.Time
	if cmd.EndDate != nil && *cmd.EndDate != "" {
		e, err := time.Parse("2006-01-02", *cmd.EndDate)
		if err != nil {
			return nil, httpx.ValidationFailed(map[string]string{"end_date": "expected YYYY-MM-DD"})
		}
		if e.Before(start) {
			return nil, httpx.ValidationFailed(map[string]string{"end_date": "must be >= start_date"})
		}
		end = &e
	}
	var reminder *time.Time
	if cmd.ReminderTime != nil && *cmd.ReminderTime != "" {
		rt, err := time.Parse("15:04", *cmd.ReminderTime)
		if err != nil {
			return nil, httpx.ValidationFailed(map[string]string{"reminder_time": "expected HH:MM"})
		}
		reminder = &rt
	}
	if !validCategory(cmd.Category) || !validFrequency(cmd.Frequency) {
		return nil, httpx.ValidationFailed(map[string]string{"category": "unknown category or frequency"})
	}

	q := db.New(s.db.Querier(ctx))
	doc, err := q.GetDoctorProfileForCarePlan(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}
	if doc.VerificationStatus != "Verified" {
		return nil, httpx.Forbidden("only verified doctors can create care plans")
	}

	if cmd.ConsultationID != nil {
		cc, err := s.consultations.ClinicalContext(ctx, *cmd.ConsultationID, userID)
		if err != nil {
			return nil, err
		}
		if cc.Status != "InProgress" && cc.Status != "Completed" {
			return nil, httpx.Unprocessable("consultation is not writable")
		}
		if cc.PatientProfileID != cmd.PatientProfileID {
			return nil, httpx.Unprocessable("consultation does not belong to this patient")
		}
		if cc.DoctorProfileID != doc.ID {
			return nil, httpx.Forbidden("not the treating doctor")
		}
	} else if _, err := q.PatientExists(ctx, cmd.PatientProfileID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("patient not found")
		}
		return nil, httpx.Internal(err)
	}

	var endDate pgtype.Date
	if end != nil {
		endDate = pgtype.Date{Time: dateOnly(*end), Valid: true}
	}
	var reminderTime pgtype.Time
	if reminder != nil {
		reminderTime = pgtype.Time{Microseconds: int64(reminder.Hour())*3_600_000_000 + int64(reminder.Minute())*60_000_000, Valid: true}
	}

	row, err := q.InsertProtocol(ctx, db.InsertProtocolParams{
		PatientProfileID: cmd.PatientProfileID,
		DoctorProfileID:  doc.ID,
		ConsultationID:   cmd.ConsultationID,
		Title:            cmd.Title,
		Description:      cmd.Description,
		Category:         db.CarePlanCategory(cmd.Category),
		Frequency:        db.CarePlanFrequency(cmd.Frequency),
		StartDate:        pgtype.Date{Time: dateOnly(start), Valid: true},
		EndDate:          endDate,
		ReminderTime:     reminderTime,
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	out := &Response{
		ID: row.ID, PatientProfileID: cmd.PatientProfileID,
		DoctorProfileID: doc.ID, ConsultationID: cmd.ConsultationID,
		Title: cmd.Title, Description: cmd.Description,
		Category: cmd.Category, Frequency: cmd.Frequency,
		StartDate: start.Format("2006-01-02"),
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
	}
	if end != nil {
		s := end.Format("2006-01-02")
		out.EndDate = &s
	}
	if reminder != nil {
		s := reminder.Format("15:04")
		out.ReminderTime = &s
	}
	return out, nil
}

func validCategory(c string) bool {
	switch careplan.Category(c) {
	case careplan.CategoryMedication, careplan.CategoryVitals, careplan.CategoryLifestyle,
		careplan.CategoryFollowUp, careplan.CategoryOther:
		return true
	}
	return false
}

func validFrequency(f string) bool {
	switch careplan.Frequency(f) {
	case careplan.FrequencyDaily, careplan.FrequencyEveryOtherDay,
		careplan.FrequencyWeekly, careplan.FrequencyOnce:
		return true
	}
	return false
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
