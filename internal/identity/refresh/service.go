package refresh

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db      *database.DB
	jwt     *auth.TokenIssuer
	refresh *auth.RefreshStore
}

func NewService(db *database.DB, jwt *auth.TokenIssuer, refresh *auth.RefreshStore) *Service {
	return &Service{db: db, jwt: jwt, refresh: refresh}
}

type Command struct {
	RefreshToken string `json:"refresh_token"`
}

type Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	var resp *Response
	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		userID, err := s.refresh.Consume(ctx, cmd.RefreshToken)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Unauthenticated("invalid refresh token")
			}
			return httpx.Internal(err)
		}

		// Fetch user to get current role.
		q := s.db.Querier(ctx)
		var role string
		if err := q.QueryRow(ctx,
			`SELECT role FROM identity_users WHERE id = $1 AND deleted_at IS NULL`,
			userID).Scan(&role); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Unauthenticated("user no longer valid")
			}
			return httpx.Internal(err)
		}

		access, err := s.jwt.IssueAccess(userID, role)
		if err != nil {
			return httpx.Internal(err)
		}
		refresh, err := s.refresh.Issue(ctx, userID)
		if err != nil {
			return httpx.Internal(err)
		}

		resp = &Response{AccessToken: access, RefreshToken: refresh, ExpiresIn: s.jwt.ExpiresIn()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
