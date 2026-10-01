package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctors/contracts"
	"tibi/internal/doctors/createsession"
	"tibi/internal/platform/httpx"
)

func (a *API) ProfileIDByUserID(ctx context.Context, userID int64) (int64, error) {
	var id int64
	err := a.db.Querier(ctx).QueryRow(ctx, `
		SELECT id FROM doctors_profiles
		WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.NotFound("doctor profile not found")
	}
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return id, nil
}

func (a *API) GetDoctorProfileID(ctx context.Context, doctorProfileID int64) (int64, error) {
	var id int64
	err := a.db.Querier(ctx).QueryRow(ctx, `
		SELECT id FROM doctors_profiles
		WHERE id = $1 AND deleted_at IS NULL`, doctorProfileID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, httpx.NotFound("doctor profile not found")
	}
	if err != nil {
		return 0, httpx.Internal(err)
	}
	return id, nil
}

func (a *API) ConsultationFee(ctx context.Context, doctorProfileID int64) (string, string, error) {
	var amount, currency string
	err := a.db.Querier(ctx).QueryRow(ctx, `
		SELECT consultation_fee::text, currency
		FROM doctors_profiles
		WHERE id = $1 AND deleted_at IS NULL`, doctorProfileID).Scan(&amount, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", httpx.NotFound("doctor profile not found")
	}
	if err != nil {
		return "", "", httpx.Internal(err)
	}
	return strings.TrimSpace(amount), strings.TrimSpace(currency), nil
}

func (a *API) MarkVerified(ctx context.Context, doctorProfileID int64) error {
	return a.setVerification(ctx, doctorProfileID, "Verified", nil)
}

func (a *API) MarkUnverified(ctx context.Context, doctorProfileID int64) error {
	return a.setVerification(ctx, doctorProfileID, "NotSubmitted", nil)
}

func (a *API) MarkRejected(ctx context.Context, doctorProfileID int64, reason string) error {
	return a.setVerification(ctx, doctorProfileID, "Rejected", &reason)
}

func (a *API) setVerification(ctx context.Context, id int64, status string, reason *string) error {
	tag, err := a.db.Querier(ctx).Exec(ctx, `
		UPDATE doctors_profiles
		SET verification_status = $2,
		    verification_rejection_reason = $3,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id, status, reason)
	if err != nil {
		return httpx.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return httpx.NotFound("doctor profile not found")
	}
	return nil
}

func (a *API) ValidateSlot(ctx context.Context, doctorProfileID int64, scheduledAt time.Time) (*contracts.SlotInfo, error) {
	info, err := a.sessions.ValidateSlot(ctx, doctorProfileID, scheduledAt)
	if err != nil {
		return nil, err
	}
	return &contracts.SlotInfo{
		BlockStartTime:      info.BlockStartTime,
		BlockEndTime:        info.BlockEndTime,
		SlotDurationMinutes: info.SlotDurationMinutes,
	}, nil
}

func (a *API) EnsureSession(ctx context.Context, doctorProfileID int64, date time.Time, info *contracts.SlotInfo) (int64, error) {
	return a.sessions.EnsureSession(ctx, doctorProfileID, date, &createsession.SlotInfo{
		BlockStartTime:      info.BlockStartTime,
		BlockEndTime:        info.BlockEndTime,
		SlotDurationMinutes: info.SlotDurationMinutes,
	})
}
