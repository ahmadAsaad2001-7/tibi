package kashier

import (
	"encoding/json"
	"errors"
	"strconv"
)

// Event is the subset of the webhook payload this system cares about.
// Additional fields are ignored. If Kashier's payload shape differs,
// adjust the struct tags — the rest of the pipeline is unchanged.
type Event struct {
	TransactionID   string `json:"transactionId"`
	MerchantOrderID string `json:"merchantOrderId"`
	Status          string `json:"status"` // "SUCCESS" | "FAILED" | "PENDING" | ...
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
}

func ParseEvent(rawBody []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(rawBody, &e); err != nil {
		return nil, err
	}
	if e.TransactionID == "" {
		return nil, errors.New("missing transactionId")
	}
	if e.MerchantOrderID == "" {
		return nil, errors.New("missing merchantOrderId")
	}
	return &e, nil
}

// ConsultationID extracts the consultation ID we encoded as the merchant
// order id at session-creation time.
func (e *Event) ConsultationID() (int64, error) {
	return strconv.ParseInt(e.MerchantOrderID, 10, 64)
}

// Succeeded reports whether the event represents a completed payment.
// Update the switch when Kashier's status vocabulary is confirmed.
func (e *Event) Succeeded() bool {
	switch e.Status {
	case "SUCCESS", "SUCCEEDED", "PAID":
		return true
	}
	return false
}

func (e *Event) Failed() bool {
	switch e.Status {
	case "FAILED", "DECLINED", "CANCELLED":
		return true
	}
	return false
}
