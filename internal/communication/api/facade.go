package api

import (
	"context"

	"tibi/internal/communication/contracts"
	"tibi/internal/communication/createnotification"
	"tibi/internal/communication/notifymessage"
)

type API struct {
	creator *createnotification.Service
	msgs    *notifymessage.Service
}

func New(creator *createnotification.Service, msgs *notifymessage.Service) *API {
	return &API{creator: creator, msgs: msgs}
}

var _ contracts.API = (*API)(nil)

func (a *API) CreateNotification(ctx context.Context, in contracts.CreateNotificationInput) (int64, error) {
	return a.creator.Create(ctx, in)
}

func (a *API) NotifyMessage(ctx context.Context, in contracts.NotifyMessageInput) error {
	return a.msgs.Execute(ctx, in)
}
