package cancelprotocol

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/clinical/careplan/cancelprotocol/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

type Response struct {
	ID          int64  `json:"id"`
	CancelledAt string `json:"cancelled_at"`
}

func (s *Service) Execute(ctx context.Context, userID, protocolID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	docID, err := q.GetDoctorProfileIDForUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.Forbidden("doctor profile not found")
		}
		return nil, httpx.Internal(err)
	}
	row, err := q.GetProtocolForCancel(ctx, protocolID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("protocol not found")
		}
		return nil, httpx.Internal(err)
	}
	if row.DoctorProfileID != docID {
		return nil, httpx.Forbidden("not your protocol")
	}
	if row.CancelledAt.Valid {
		return nil, &httpx.APIError{Status: 409, Code: "already_cancelled", Message: "protocol is already cancelled"}
	}
	now := s.clock()
	if err := q.CancelProtocol(ctx, db.CancelProtocolParams{
		ID:          row.ID,
		CancelledAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{ID: row.ID, CancelledAt: now.Format(time.RFC3339)}, nil
}
