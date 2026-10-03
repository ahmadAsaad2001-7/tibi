package addattachment

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/clinical/addattachment/db"
	consultationscontracts "tibi/internal/consultations/contracts"
	filescontracts "tibi/internal/files/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db            *database.DB
	consultations consultationscontracts.API
	files         filescontracts.API
}

func NewService(db *database.DB, c consultationscontracts.API, f filescontracts.API) *Service {
	return &Service{db: db, consultations: c, files: f}
}

type Command struct {
	FileID int64   `json:"file_id" validate:"required,gt=0"`
	Label  *string `json:"label"   validate:"omitempty,max=200"`
}

type Response struct {
	ID        int64     `json:"id"`
	FileID    int64     `json:"file_id"`
	Label     *string   `json:"label,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64, cmd Command) (*Response, error) {
	cc, err := s.consultations.ClinicalContext(ctx, consultationID, userID)
	if err != nil {
		return nil, err
	}
	if cc.Status != "InProgress" && cc.Status != "Completed" {
		return nil, httpx.Unprocessable("consultation is not writable")
	}

	// The file must belong to the doctor adding it. This stops doctor A
	// from attaching doctor B's uploads.
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
	if meta.Scope != "MedicalAttachment" {
		return nil, httpx.ValidationFailed(map[string]string{
			"file_id": "file scope must be MedicalAttachment",
		})
	}

	var resp *Response
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		record, err := q.GetRecordForAttachment(ctx, consultationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("medical record not yet created for this consultation")
			}
			return httpx.Internal(err)
		}
		row, err := q.InsertAttachment(ctx, db.InsertAttachmentParams{
			MedicalRecordID: record.ID,
			FileID:          cmd.FileID,
			Label:           cmd.Label,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		resp = &Response{
			ID:        row.ID,
			FileID:    cmd.FileID,
			Label:     cmd.Label,
			CreatedAt: row.CreatedAt.Time,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
