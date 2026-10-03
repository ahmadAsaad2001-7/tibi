package api

import (
	"context"

	"tibi/internal/clinical/contracts"
	"tibi/internal/clinical/getprescription"
	"tibi/internal/clinical/getrecord"
)

type API struct {
	records       *getrecord.Service
	prescriptions *getprescription.Service
}

func New(records *getrecord.Service, prescriptions *getprescription.Service) *API {
	return &API{records: records, prescriptions: prescriptions}
}

var _ contracts.API = (*API)(nil)

func (a *API) HasRecord(ctx context.Context, consultationID int64) (bool, error) {
	// cheap existence query; add when a consumer needs it
	return false, nil
}
