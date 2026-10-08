package submitmessage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctorposts/freemessages"
	"tibi/internal/doctorposts/freemessages/submitmessage/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ratelimit"
)

type Service struct {
	db      *database.DB
	ipLimit ratelimit.Limiter
	emLimit ratelimit.Limiter
	clock   func() time.Time
}

func NewService(db *database.DB, ipLimit, emLimit ratelimit.Limiter) *Service {
	return &Service{db: db, ipLimit: ipLimit, emLimit: emLimit, clock: time.Now}
}

type Response struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	// Rate limit before any DB work. Two buckets: per IP, per email.
	// Errors fail open (SD36) via ratelimit.AllowOrFailOpen.
	if !ratelimit.AllowOrFailOpen(ctx, s.ipLimit, cmd.SenderIP) {
		return nil, httpx.ErrorRateLimited("too many messages from this network; retry later")
	}
	if !ratelimit.AllowOrFailOpen(ctx, s.emLimit, cmd.SenderEmail) {
		return nil, httpx.ErrorRateLimited("too many messages from this email; retry later")
	}

	msg, err := freemessages.New(
		cmd.DoctorProfileID,
		cmd.SenderName,
		cmd.SenderEmail,
		cmd.SenderPhone,
		cmd.Content,
		cmd.SenderIP,
		s.clock(),
	)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"message": err.Error()})
	}

	var resp *Response
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		if _, err := q.GetDoctorForFreeMessage(ctx, cmd.DoctorProfileID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("doctor not found")
			}
			return httpx.Internal(err)
		}

		row, err := q.InsertFreeMessage(ctx, db.InsertFreeMessageParams{
			DoctorProfileID: msg.DoctorProfileID,
			SenderName:      msg.SenderName,
			SenderEmail:     msg.SenderEmail,
			SenderPhone:     msg.SenderPhone,
			Content:         msg.Content,
			SenderIp:        pgInet(cmd.SenderIP),
		})
		if err != nil {
			return httpx.Internal(err)
		}
		resp = &Response{ID: row.ID, CreatedAt: row.CreatedAt.Time}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
