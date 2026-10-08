package requestpasswordreset

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/identity/onetimetoken"
	"tibi/internal/identity/requestpasswordreset/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/email"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ratelimit"
)

const resetTTL = time.Hour

type Service struct {
	db         *database.DB
	mailer     email.Sender
	ipLimit    ratelimit.Limiter
	emLimit    ratelimit.Limiter
	appBaseURL string
	clock      func() time.Time
}

func NewService(
	db *database.DB,
	mailer email.Sender,
	ipLimit, emLimit ratelimit.Limiter,
	appBaseURL string,
) *Service {
	return &Service{
		db: db, mailer: mailer,
		ipLimit: ipLimit, emLimit: emLimit,
		appBaseURL: appBaseURL, clock: time.Now,
	}
}

// Response is deliberately uniform. Same shape whether the email exists or
// not (SD33).
type Response struct {
	Message string `json:"message"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	if !ratelimit.AllowOrFailOpen(ctx, s.ipLimit, cmd.ClientIP) {
		return nil, httpx.ErrorRateLimited("too many requests")
	}
	if !ratelimit.AllowOrFailOpen(ctx, s.emLimit, cmd.Email) {
		return nil, httpx.ErrorRateLimited("too many requests")
	}

	uniform := &Response{
		Message: "If an account exists for that email, a reset link was sent.",
	}

	var rawToken string
	var recipient string

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetUserByEmailForReset(ctx, cmd.Email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Silent: return the same response. No timing leak because
				// the DB call above ran either way.
				return nil
			}
			return httpx.Internal(err)
		}

		if err := q.InvalidateOpenPasswordResets(ctx, row.ID); err != nil {
			return httpx.Internal(err)
		}

		tok, err := onetimetoken.New(resetTTL, s.clock())
		if err != nil {
			return httpx.Internal(err)
		}

		if err := q.InsertPasswordReset(ctx, db.InsertPasswordResetParams{
			UserID:    row.ID,
			TokenHash: tok.Hash,
			ExpiresAt: pgTime(tok.ExpiresAt),
		}); err != nil {
			return httpx.Internal(err)
		}

		rawToken = tok.Raw
		recipient = row.Email
		return nil
	})
	if err != nil {
		return nil, err
	}

	// After commit: best-effort email. Failure does not fail the request
	// (SD28 convention).
	if rawToken != "" && s.mailer != nil {
		link := s.appBaseURL + "/reset-password?token=" + rawToken
		body := "Click the link below to reset your password. This link expires in 1 hour.\n\n" + link
		_ = s.mailer.Send(ctx, recipient, "Reset your password", body)
	}

	return uniform, nil
}