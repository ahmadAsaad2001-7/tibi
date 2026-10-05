package createpost

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctorposts/createpost/db"
	doctorscontracts "tibi/internal/doctors/contracts"
	filescontracts "tibi/internal/files/contracts"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db      *database.DB
	doctors doctorscontracts.API
	files   filescontracts.API
	clock   func() time.Time
}

// ✅ FIX 1: إزالة تعريف NewService المكرر (كان موجوداً مرتين)
func NewService(db *database.DB, doctors doctorscontracts.API, files filescontracts.API) *Service {
	return &Service{db: db, doctors: doctors, files: files, clock: time.Now}
}

type Command struct {
	Title         string  `json:"title"           validate:"required,min=1,max=200"`
	Content       string  `json:"content"         validate:"required,min=1"`
	Excerpt       *string `json:"excerpt"         validate:"omitempty,max=500"`
	Type          string  `json:"type"            validate:"required,oneof=HealthTip PatientEducation ClinicNews Publication"`
	CoverImageURL *string `json:"cover_image_url" validate:"omitempty,url"`
	AttachmentIDs []int64 `json:"attachment_ids"  validate:"omitempty,max=10,dive,gt=0"`
}

type Response struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Excerpt     *string   `json:"excerpt,omitempty"`
	Type        string    `json:"type"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) Execute(ctx context.Context, userID int64, cmd Command) (*Response, error) {
	// ✅ FIX 2: استخدام db.PostType مباشرة. تم إزالة .Valid() لأن validate tag يتحقق من ذلك مسبقاً
	postType := db.PostType(cmd.Type)

	doctorProfileID, err := s.doctors.ProfileIDByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate attachments before opening the transaction.
	type verified struct {
		FileID int64
	}
	verifiedAttachments := make([]verified, 0, len(cmd.AttachmentIDs))
	seen := map[int64]struct{}{}
	for _, fid := range cmd.AttachmentIDs {
		if _, dup := seen[fid]; dup {
			return nil, httpx.ValidationFailed(map[string]string{
				"attachment_ids": "duplicate file id",
			})
		}
		seen[fid] = struct{}{}

		meta, err := s.files.Get(ctx, fid)
		if err != nil {
			if errors.Is(err, filescontracts.ErrNotFound) {
				return nil, httpx.ValidationFailed(map[string]string{
					"attachment_ids": "one or more files not found",
				})
			}
			return nil, httpx.Internal(err)
		}
		if meta.UploaderID != userID {
			return nil, httpx.NotFound("file not found")
		}
		if meta.Scope != "PostAttachment" {
			return nil, httpx.ValidationFailed(map[string]string{
				"attachment_ids": "file scope must be PostAttachment",
			})
		}
		verifiedAttachments = append(verifiedAttachments, verified{FileID: fid})
	}

	// ✅ FIX 3: تحويل *string إلى pgtype.Text
	var excerpt pgtype.Text
	if cmd.Excerpt != nil {
		excerpt = pgtype.Text{String: *cmd.Excerpt, Valid: true}
	}

	var coverImageURL pgtype.Text
	if cmd.CoverImageURL != nil {
		coverImageURL = pgtype.Text{String: *cmd.CoverImageURL, Valid: true}
	}

	var resp *Response
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		row, err := q.InsertPost(ctx, db.InsertPostParams{
			DoctorProfileID: doctorProfileID,
			Title:           cmd.Title,
			Content:         cmd.Content,
			Excerpt:         excerpt,       // ✅ FIX 4: استخدام المتغير المحول
			Type:            postType,      // ✅ FIX 5: استخدام db.PostType مباشرة (ليس string)
			CoverImageUrl:   coverImageURL, // ✅ FIX 6: استخدام المتغير المحول
		})
		if err != nil {
			return httpx.Internal(err)
		}

		for _, a := range verifiedAttachments {
			fid := a.FileID
			// ✅ FIX 7: استخدام pgtype.Int8 لـ FileID بناءً على هيكل ContentPostAttachment المولد
			if err := q.InsertPostAttachment(ctx, db.InsertPostAttachmentParams{
				DoctorPostID: row.ID,
				FileID:       pgtype.Int8{Int64: fid, Valid: true},
			}); err != nil {
				return httpx.Internal(err)
			}
		}

		resp = &Response{
			ID:          row.ID,
			Title:       cmd.Title,
			Content:     cmd.Content,
			Excerpt:     cmd.Excerpt,
			Type:        string(postType),
			IsPublished: false,
			CreatedAt:   row.CreatedAt.Time,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}
