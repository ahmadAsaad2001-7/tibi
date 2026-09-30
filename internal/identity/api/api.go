package api

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/identity/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type API struct {
	db *database.DB
}

func New(db *database.DB) *API { return &API{db: db} }

var _ contracts.API = (*API)(nil)

func (a *API) PromotePendingDoctorToDoctor(ctx context.Context, userID int64) error {
	return a.setRole(ctx, userID, "PendingDoctor", "Doctor")
}

func (a *API) DemoteDoctorToPending(ctx context.Context, userID int64) error {
	return a.setRole(ctx, userID, "Doctor", "PendingDoctor")
}

// setRole is a guarded transition. It updates role only if the current
// role matches the expected "from" value, returning a domain-specific
// error otherwise. This is the identity-side equivalent of an
// optimistic concurrency check.
func (a *API) setRole(ctx context.Context, userID int64, from, to string) error {
	q := a.db.Querier(ctx)
	tag, err := q.Exec(ctx,
		`UPDATE identity_users
		    SET role = $1, updated_at = now()
		  WHERE id = $2 AND role = $3 AND deleted_at IS NULL`,
		to, userID, from)
	if err != nil {
		return httpx.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		// Either the user does not exist, or the role was not what we expected.
		// Distinguish: read the current role.
		var current string
		err := q.QueryRow(ctx,
			`SELECT role FROM identity_users WHERE id = $1 AND deleted_at IS NULL`,
			userID).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NotFound("user not found")
		}
		if err != nil {
			return httpx.Internal(err)
		}
		return httpx.Conflict("user role is " + current + ", expected " + from)
	}
	return nil
}
