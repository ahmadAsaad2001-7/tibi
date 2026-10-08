package liftsuspension

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/admin/liftsuspension/db"
	"tibi/internal/admin/suspension"
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
	ID       int64     `json:"id"`
	LiftedAt time.Time `json:"lifted_at"`
}

func (s *Service) Execute(ctx context.Context, adminID, suspensionID int64, cmd Command) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetSuspensionForUpdate(ctx, suspensionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("suspension not found")
			}
			return httpx.Internal(err)
		}

		sus := &suspension.Suspension{
			ID:       row.ID,
			UserID:   row.UserID,
			Reason:   row.Reason,
			FromTS:   row.FromTs.Time,
			ToTS:     tsPtr(row.ToTs),
			LiftedAt: tsPtr(row.LiftedAt),
		}

		now := s.clock()
		if err := sus.Lift(adminID, cmd.Reason, now); err != nil {
			if errors.Is(err, suspension.ErrAlreadyLifted) {
				return httpx.Conflict("suspension is already lifted")
			}
			return httpx.ValidationFailed(map[string]string{"reason": err.Error()})
		}

		n, err := q.LiftSuspension(ctx, db.LiftSuspensionParams{
			ID:         sus.ID,
			LiftedAt:   pgTime(now),
			LiftedBy:   &adminID,
			LiftReason: sus.LiftReason,
			Xmin:       xminToUint32(row.Xmin),
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("suspension was modified concurrently")
		}

		resp = &Response{ID: sus.ID, LiftedAt: now}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func tsPtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}

// xminToUint32 converts the string xmin into the pgtype.Uint32 the update
// param expects. Invalid/empty yields an invalid value, making the optimistic
// guard a no-op (0 rows -> conflict).
func xminToUint32(x string) pgtype.Uint32 {
	if x == "" {
		return pgtype.Uint32{}
	}
	val, err := strconv.ParseUint(x, 10, 32)
	if err != nil {
		return pgtype.Uint32{}
	}
	return pgtype.Uint32{Uint32: uint32(val), Valid: true}
}