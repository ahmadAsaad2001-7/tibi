package requestemailverification

import (
	"context"
	"strconv"
	"time"

	"tibi/internal/identity/onetimetoken"
	"tibi/internal/identity/requestemailverification/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/email"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ratelimit"
)

const verifyTTL = 24 * time.Hour

type Service struct {
	db         *database.DB
	mailer     email.Sender
	userLimit  ratelimit.Limiter
	appBaseURL string
	clock      func() time.Time
}

func NewService(
	db *database.DB,
	mailer email.Sender,
	userLimit ratelimit.Limiter,
	appBaseURL string,
) *Service {
	return &Service{
		db: db, mailer: mailer,
		userLimit:  userLimit,
		appBaseURL: appBaseURL, clock: time.Now,
	}
}

// Execute sends a verification email for the given (authenticated) user.
// It satisfies register.EmailVerifier via RequestVerification.
func (s *Service) Execute(ctx context.Context, userID int64) error {
	return s.RequestVerification(ctx, userID)
}

// RequestVerification implements register.EmailVerifier. Same operation as
// Execute, exposed under the name register depends on.
func (s *Service) RequestVerification(ctx context.Context, userID int64) error {
	// SD32: 3 per hour per user ID. The key is namespaced so it never
	// collides with an email/IP bucket.
	key := "user:" + strconv.FormatInt(userID, 10)
	if !ratelimit.AllowOrFailOpen(ctx, s.userLimit, key) {
		// Best-effort: fail open on rate limit too. A logged-in user hammering
		// "resend" must not get a hard error; nothing sensitive is revealed.
		return nil
	}

	var rawToken string
	var recipient string
	var alreadyVerified bool

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetUserForVerification(ctx, userID)
		if err != nil {
			return httpx.Internal(err)
		}
		recipient = row.Email

		// Idempotent: if already verified, do not send another email.
		if row.EmailVerifiedAt.Valid {
			alreadyVerified = true
			return nil
		}

		if err := q.InvalidateOpenVerifications(ctx, row.ID); err != nil {
			return httpx.Internal(err)
		}

		tok, err := onetimetoken.New(verifyTTL, s.clock())
		if err != nil {
			return httpx.Internal(err)
		}

		if err := q.InsertEmailVerification(ctx, db.InsertEmailVerificationParams{
			UserID:    row.ID,
			TokenHash: tok.Hash,
			ExpiresAt: pgTime(tok.ExpiresAt),
		}); err != nil {
			return httpx.Internal(err)
		}

		rawToken = tok.Raw
		return nil
	})
	if err != nil {
		return err
	}

	if !alreadyVerified && rawToken != "" && s.mailer != nil {
		link := s.appBaseURL + "/verify-email?token=" + rawToken
		body := "Click the link below to verify your email address. This link expires in 24 hours.\n\n" + link
		_ = s.mailer.Send(ctx, recipient, "Verify your email", body)
	}

	return nil
}
