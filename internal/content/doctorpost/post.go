package doctorpost

import (
	"errors"
	"time"
)

type PostType string

const (
	TypeHealthTip        PostType = "HealthTip"
	TypePatientEducation PostType = "PatientEducation"
	TypeClinicNews       PostType = "ClinicNews"
	TypePublication      PostType = "Publication"
)

func (t PostType) Valid() bool {
	switch t {
	case TypeHealthTip, TypePatientEducation, TypeClinicNews, TypePublication:
		return true
	}
	return false
}

type Post struct {
	ID              int64
	DoctorProfileID int64
	Title           string
	Content         string
	Excerpt         *string
	Type            PostType
	CoverImageURL   *string
	ViewCount       int
	LikeCount       int
	IsPublished     bool
	IsFeatured      bool
	PublishedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time

	Attachments []Attachment
}

type Attachment struct {
	ID       int64
	FileURL  string
	FileType string
}

var (
	ErrAlreadyPublished = errors.New("post is already published")
	ErrNotPublished     = errors.New("post is not published")
)

func (p *Post) Publish(now time.Time) error {
	if p.IsPublished {
		return ErrAlreadyPublished
	}
	p.IsPublished = true
	p.PublishedAt = &now
	p.UpdatedAt = now
	return nil
}

func (p *Post) Unpublish(now time.Time) error {
	if !p.IsPublished {
		return ErrNotPublished
	}
	p.IsPublished = false
	p.PublishedAt = nil
	p.UpdatedAt = now
	return nil
}

// Update applies fields from an update command. Nil pointer fields mean
// "not present, do not change."
func (p *Post) Update(in UpdateInput, now time.Time) {
	if in.Title != nil {
		p.Title = *in.Title
	}
	if in.Content != nil {
		p.Content = *in.Content
	}
	if in.Excerpt != nil {
		p.Excerpt = *in.Excerpt
	}
	if in.Type != nil {
		p.Type = *in.Type
	}
	if in.CoverImageURL != nil {
		p.CoverImageURL = *in.CoverImageURL
	}
	p.UpdatedAt = now
}

type UpdateInput struct {
	Title         *string
	Content       *string
	Excerpt       **string
	Type          *PostType
	CoverImageURL **string
}
