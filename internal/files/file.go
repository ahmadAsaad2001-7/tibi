package file

import "time"

type Scope string

const (
	ScopeProfileImage      Scope = "ProfileImage"
	ScopePostAttachment    Scope = "PostAttachment"
	ScopeMedicalAttachment Scope = "MedicalAttachment"
)

func (s Scope) Valid() bool {
	switch s {
	case ScopeProfileImage, ScopePostAttachment, ScopeMedicalAttachment:
		return true
	}
	return false
}

// File is the aggregate. Object bytes live in object storage; this struct
// is metadata only. The state machine is intentionally minimal: rows are
// either live or soft-deleted. There is no "uploading" state because the
// object is written before the row is inserted.
type File struct {
	ID            int64
	UploaderID    int64
	Scope         Scope
	ObjectKey     string
	OriginalName  string
	ContentType   string
	SizeBytes     int64
	ContentSHA256 string
	CreatedAt     time.Time
	DeletedAt     *time.Time
}

func (f *File) IsDeleted() bool { return f.DeletedAt != nil }
