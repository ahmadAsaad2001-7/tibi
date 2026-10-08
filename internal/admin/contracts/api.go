// Package contracts is what other modules may import from Admin. Currently,
// only the Identity login slice needs it, to check suspension status before
// issuing tokens (R29, SD49).
package contracts

import (
	"context"
	"time"
)

// API is the Admin surface exposed to other modules.
type API interface {
	// ActiveSuspension returns the user's currently active suspension,
	// or (nil, nil) if none.
	ActiveSuspension(ctx context.Context, userID int64) (*ActiveSuspension, error)
}

type ActiveSuspension struct {
	Reason string
	From   time.Time
	Until  *time.Time // nil = permanent
}