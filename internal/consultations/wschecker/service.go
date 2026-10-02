package wschecker

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tibi/internal/consultations/wschecker/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

func (s *Service) IsConsultationMember(ctx context.Context, userID, consultationID int64) (bool, error) {
	q := db.New(s.db.Querier(ctx))
	ok, err := q.IsConsultationMember(ctx, db.IsConsultationMemberParams{
		ID:     consultationID,
		UserID: userID,
	})
	if err != nil {
		return false, httpx.Internal(err)
	}
	return ok, nil
}

func (s *Service) OtherConsultationMember(ctx context.Context, userID, consultationID int64) (int64, error) {
	q := db.New(s.db.Querier(ctx))
	other, err := q.OtherConsultationMember(ctx, db.OtherConsultationMemberParams{
		ID:     consultationID,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, httpx.Internal(err)
	}
	return other, nil
}
