package setprofileimage

import (
	"context"
	"errors"

	filescontracts "tibi/internal/files/contracts"
	"tibi/internal/identity/setprofileimage/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db    *database.DB
	files filescontracts.API
}

func NewService(db *database.DB, files filescontracts.API) *Service {
	return &Service{db: db, files: files}
}

type Response struct {
	ProfileImageURL string `json:"profile_image_url"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	meta, err := s.files.Get(ctx, cmd.FileID)
	if err != nil {
		if errors.Is(err, filescontracts.ErrNotFound) {
			return nil, httpx.NotFound("file not found")
		}
		return nil, httpx.Internal(err)
	}
	if meta.UploaderID != userID {
		return nil, httpx.NotFound("file not found")
	}
	if meta.Scope != "ProfileImage" {
		return nil, httpx.ValidationFailed(map[string]string{
			"file_id": "file scope must be ProfileImage",
		})
	}

	if err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))
		return q.SetProfileImageFile(ctx, db.SetProfileImageFileParams{
			ID:                 userID,
			ProfileImageFileID: &cmd.FileID,
		})
	}); err != nil {
		return nil, httpx.Internal(err)
	}

	url, err := s.files.PresignGet(ctx, cmd.FileID, 86400)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{ProfileImageURL: url}, nil
}
