package contracts

import "context"

type API interface {
	HasRecord(ctx context.Context, consultationID int64) (bool, error)
}
