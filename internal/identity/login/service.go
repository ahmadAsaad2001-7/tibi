package login

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/identity/login/db"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db      *database.DB
	hasher  *auth.PasswordHasher
	jwt     *auth.TokenIssuer
	refresh *auth.RefreshStore
}

func NewService(db *database.DB, hasher *auth.PasswordHasher, jwt *auth.TokenIssuer, refresh *auth.RefreshStore) *Service {
	return &Service{db: db, hasher: hasher, jwt: jwt, refresh: refresh}
}

type Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))
		row, err := q.FindUserByEmail(ctx, cmd.Email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Same message for "no such user" and "wrong password".
				return httpx.Unauthenticated("invalid credentials")
			}
			return httpx.Internal(err)
		}

		ok, err := s.hasher.Verify(cmd.Password, row.PasswordHash)
		if err != nil || !ok {
			return httpx.Unauthenticated("invalid credentials")
		}

		access, err := s.jwt.IssueAccess(row.ID, string(row.Role))
		if err != nil {
			return httpx.Internal(err)
		}
		refresh, err := s.refresh.Issue(ctx, row.ID)
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
