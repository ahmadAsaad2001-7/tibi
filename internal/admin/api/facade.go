package api

import (
	"context"

	"tibi/internal/admin/activecheck"
	"tibi/internal/admin/contracts"
)

type API struct {
	active *activecheck.Service
}

func New(active *activecheck.Service) *API {
	return &API{active: active}
}

var _ contracts.API = (*API)(nil)

func (a *API) ActiveSuspension(ctx context.Context, userID int64) (*contracts.ActiveSuspension, error) {
	return a.active.ActiveSuspension(ctx, userID)
}