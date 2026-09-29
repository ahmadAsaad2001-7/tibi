package logout

import (
	"context"

	"github.com/ahmadAsaad2001-7/tibi/internal/platform/auth"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/httpx"
)

type Service struct {
	refresh *auth.RefreshStore
}

func NewService(refresh *auth.RefreshStore) *Service {
	return &Service{refresh: refresh}
}

type Command struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) error {
	if err := s.refresh.Revoke(ctx, cmd.RefreshToken); err != nil {
		return httpx.Internal(err)
	}
	return nil
}
