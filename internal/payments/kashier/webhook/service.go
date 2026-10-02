package webhook

import (
	"context"
	"errors"
	"tibi/internal/payments/payment"
	"time"

	"github.com/jackc/pgx/v5"

	consultationscontracts "tibi/internal/consultations/contracts"
	"tibi/internal/payments/kashier"
	"tibi/internal/payments/kashier/webhook/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db            *database.DB
	secret        string
	consultations consultationscontracts.API
	clock         func() time.Time
}

func NewService(db *database.DB, secret string, c consultationscontracts.API) *Service {
	return &Service{db: db, secret: secret, consultations: c, clock: time.Now}
}

// Execute handles a webhook. Returns nil on success or idempotent replay.
func (s *Service) Execute(ctx context.Context, rawBody []byte, signature string) error {
	if !kashier.VerifySignature(s.secret, rawBody, signature) {
		return httpx.Unauthenticated("invalid signature")
	}

	event, err := kashier.ParseEvent(rawBody)
	if err != nil {
		return httpx.BadRequest("malformed payload")
	}

	consultationID, err := event.ConsultationID()
	if err != nil {
		return httpx.BadRequest("invalid merchant order id")
	}

	// Idempotency: has this transaction already been processed?
	if seen, err := s.paymentExists(ctx, event.TransactionID); err != nil {
		return err
	} else if seen {
		return nil
	}

	return s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.GetPaymentByConsultationForUpdate(ctx, consultationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Webhook references a consultation we do not know about.
				// Log and return 400; do not let Kashier retry.
				return httpx.BadRequest("unknown consultation")
			}
			return httpx.Internal(err)
		}

		p := &payment.Payment{
			ID:               row.ID,
			PatientProfileID: row.PatientProfileID,
			DoctorProfileID:  row.DoctorProfileID,
			ConsultationID:   row.ConsultationID,
			Amount:           row.Amount,
			Currency:         row.Currency,
			Channel:          payment.Channel(row.Channel),
			TransactionID:    row.TransactionId,
			Status:           payment.Status(row.Status),
			TransactionDate:  tsPtr(row.TransactionDate),
			CheckoutURL:      row.CheckoutUrl,
			CreatedAt:        row.CreatedAt.Time,
		}

		now := s.clock()
		var newStatus payment.Status

		switch {
		case event.Succeeded():
			if err := p.MarkAsSucceeded(event.TransactionID, now); err != nil {
				// Already terminal. Idempotent: return nil.
				if errors.Is(err, payment.ErrInvalidTransition) {
					return nil
				}
				return httpx.Internal(err)
			}
			newStatus = payment.StatusSucceeded

		case event.Failed():
			if err := p.MarkAsFailed(event.TransactionID, now); err != nil {
				if errors.Is(err, payment.ErrInvalidTransition) {
					return nil
				}
				return httpx.Internal(err)
			}
			newStatus = payment.StatusFailed

		default:
			// Unknown status; log and accept without state change.
			return nil
		}

		txID := event.TransactionID
		n, err := q.UpdatePaymentOutcome(ctx, db.UpdatePaymentOutcomeParams{
			ID:              p.ID,
			Status:          string(newStatus),
			TransactionID:   &txID,
			TransactionDate: pgTime(now),
			Xmin:            row.Xmin,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		if n == 0 {
			return httpx.Conflict("payment was modified concurrently")
		}

		if newStatus == payment.StatusSucceeded {
			if err := s.consultations.Confirm(ctx, consultationID); err != nil {
				return err
			}
		}
		// On failure: consultation stays Pending. Patient may retry the
		// payment via a future endpoint, or the 15-minute slot hold expires.

		return nil
	})
}

func (s *Service) paymentExists(ctx context.Context, txID string) (bool, error) {
	q := db.New(s.db.Querier(ctx))
	ok, err := q.PaymentExistsByTransactionID(ctx, &txID)
	if err != nil {
		return false, httpx.Internal(err)
	}
	return ok, nil
}
