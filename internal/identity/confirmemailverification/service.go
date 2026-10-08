package confirmemailverification

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/identity/confirmemailverification/db"
	"tibi/internal/identity/onetimetoken"
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
	Message string `json:"message"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	hash := onetimetoken.HashToken(cmd.Token)

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetVerificationForUpdate(ctx, hash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("invalid or expired token")
			}
			return httpx.Internal(err)
		}

		now := s.clock()
		if row.ConsumedAt.Valid {
			return httpx.Conflict("token has already been used")
		}
		if now.After(row.ExpiresAt.Time) {
			return httpx.NotFound("invalid or expired token")
		}

		affected, err := q.ConsumeVerification(ctx, db.ConsumeVerificationParams{
			ID:   row.ID,
			Xmin: xminToUint32(row.Xmin),
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if affected == 0 {
			return httpx.Conflict("token was already consumed concurrently")
		}

		// Idempotent at the DB level via the AND email_verified_at IS NULL.
		if err := q.SetEmailVerified(ctx, row.UserID); err != nil {
			return httpx.Internal(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Response{Message: "Email verified."}, nil
}

// xminToUint32 converts the string xmin read via `xmin::text AS xmin` into the
// pgtype.Uint32 the update param expects. See confirmpasswordreset for the
// reasoning.
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