package payment

import (
	"errors"
	"time"
)

type Status string

const (
	StatusPending   Status = "Pending"
	StatusSucceeded Status = "Succeeded"
	StatusFailed    Status = "Failed"
	StatusRefunded  Status = "Refunded"
)

type Channel string

const (
	ChannelCard         Channel = "Card"
	ChannelMobileWallet Channel = "MobileWallet"
	ChannelInstaPay     Channel = "InstaPay"
)

type Payment struct {
	ID               int64
	PatientProfileID int64
	DoctorProfileID  int64
	ConsultationID   int64
	Amount           string
	Currency         string
	Channel          Channel
	TransactionID    *string
	Status           Status
	TransactionDate  *time.Time
	CheckoutURL      *string
	CreatedAt        time.Time
}

// MarkAsSucceeded is the state transition applied by the webhook (slice 7).
func (p *Payment) MarkAsSucceeded(transactionID string, now time.Time) error {
	if p.Status != StatusPending {
		return ErrInvalidTransition
	}
	p.Status = StatusSucceeded
	p.TransactionID = &transactionID
	p.TransactionDate = &now
	return nil
}

// MarkAsFailed is the state transition applied by the webhook on a failed
// payment. A Pending payment may move to Failed; terminal states may not.
func (p *Payment) MarkAsFailed(transactionID string, now time.Time) error {
	if p.Status != StatusPending {
		return ErrInvalidTransition
	}
	p.Status = StatusFailed
	p.TransactionID = &transactionID
	p.TransactionDate = &now
	return nil
}

var ErrInvalidTransition = errors.New("invalid payment status transition")
