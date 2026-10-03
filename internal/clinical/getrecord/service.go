package getrecord

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/clinical/getrecord/db"
	filescontracts "tibi/internal/files/contracts"
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

type AttachmentOut struct {
	ID        int64     `json:"id"`
	FileID    int64     `json:"file_id"`
	Label     *string   `json:"label,omitempty"`
	FileURL   string    `json:"file_url"`
	FileName  string    `json:"file_name"`
	FileSize  int64     `json:"file_size"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Response struct {
	ID                 int64           `json:"id"`
	ConsultationID     int64           `json:"consultation_id"`
	Allergies          *string         `json:"allergies"`
	CurrentMedications *string         `json:"current_medications"`
	PastConditions     *string         `json:"past_conditions"`
	DoctorNotes        *string         `json:"doctor_notes"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	Attachments        []AttachmentOut `json:"attachments"`
}

func (s *Service) Execute(ctx context.Context, userID, consultationID int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	row, err := q.GetRecordForRead(ctx, db.GetRecordForReadParams{
		ConsultationID: consultationID,
		UserID:         userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("medical record not found")
		}
		return nil, httpx.Internal(err)
	}

	atts, err := q.ListAttachments(ctx, row.ID)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	resp := &Response{
		ID:                 row.ID,
		ConsultationID:     row.ConsultationID,
		Allergies:          row.Allergies,
		CurrentMedications: row.CurrentMedications,
		PastConditions:     row.PastConditions,
		DoctorNotes:        row.DoctorNotes,
		CreatedAt:          row.CreatedAt.Time,
		UpdatedAt:          row.UpdatedAt.Time,
		Attachments:        make([]AttachmentOut, 0, len(atts)),
	}

	for _, a := range atts {
		url, err := s.files.PresignGet(ctx, a.FileID, 86400)
		if err != nil {
			return nil, httpx.Internal(err)
		}
		meta, err := s.files.Get(ctx, a.FileID)
		if err != nil {
			return nil, httpx.Internal(err)
		}
		resp.Attachments = append(resp.Attachments, AttachmentOut{
			ID:        a.ID,
			FileID:    a.FileID,
			Label:     a.Label,
			FileURL:   url,
			FileName:  meta.OriginalName,
			FileSize:  meta.SizeBytes,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
	}

	return resp, nil
}
