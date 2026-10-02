package api

import (
	"context"

	"tibi/internal/consultations/checkininfo"
	"tibi/internal/consultations/contracts"
	"tibi/internal/consultations/markcompleted"
	"tibi/internal/consultations/markconfirmed"
)

type API struct {
	markConfirmed *markconfirmed.Service
	checkInInfo   *checkininfo.Service
	markCompleted *markcompleted.Service
}

func New(markConfirmed *markconfirmed.Service, checkInInfo *checkininfo.Service, markCompleted *markcompleted.Service) *API {
	return &API{markConfirmed: markConfirmed, checkInInfo: checkInInfo, markCompleted: markCompleted}
}

var _ contracts.API = (*API)(nil)

func (a *API) Confirm(ctx context.Context, consultationID int64) error {
	return a.markConfirmed.Execute(ctx, consultationID)
}

func (a *API) GetForCheckIn(ctx context.Context, consultationID, userID int64) (*contracts.CheckInInfo, error) {
	return a.checkInInfo.GetForCheckIn(ctx, consultationID, userID)
}

func (a *API) MarkCompleted(ctx context.Context, consultationID int64) error {
	return a.markCompleted.Execute(ctx, consultationID)
}
