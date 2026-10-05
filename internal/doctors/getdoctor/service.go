package getdoctor

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/availability"
	"tibi/internal/doctors/doctorprofile"
	"tibi/internal/doctors/getdoctor/db"
	"tibi/internal/doctors/scheduleexception"
	"tibi/internal/doctors/weeklyschedule"
	filescontracts "tibi/internal/files/contracts" // ✅ Added
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

const horizonDays = 30

type Service struct {
	db    *database.DB
	files filescontracts.API // ✅ Added as per Slice 12b
	clock func() time.Time
}

// ✅ Updated to accept filesAPI
func NewService(db *database.DB, files filescontracts.API) *Service {
	return &Service{db: db, files: files, clock: time.Now}
}

type Specialty struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type WeeklyBlock struct {
	DayOfWeek           int    `json:"day_of_week"`
	Start               string `json:"start"`
	End                 string `json:"end"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
}

type Post struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Excerpt     string    `json:"excerpt"`
	Type        string    `json:"type"`
	PublishedAt time.Time `json:"published_at"`
}

type Response struct {
	ID                   int64         `json:"id"`
	FullName             string        `json:"full_name"`
	Bio                  string        `json:"bio"`
	ClinicName           string        `json:"clinic_name"`
	ClinicAddress        string        `json:"clinic_address"`
	ConsultationFee      string        `json:"consultation_fee"`
	Currency             string        `json:"currency"`
	AverageRating        string        `json:"average_rating"`
	RatingCount          int           `json:"rating_count"`
	MedicalLicenseNumber *string       `json:"medical_license_number"`
	ProfileImageURL      *string       `json:"profile_image_url"`
	Specialties          []Specialty   `json:"specialties"`
	WeeklySchedule       []WeeklyBlock `json:"weekly_schedule"`
	NextAvailableAt      *time.Time    `json:"next_available_at"`
	RecentPosts          []Post        `json:"recent_posts"`
}

func (s *Service) Execute(ctx context.Context, id int64) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	doc, err := q.GetDoctorByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NotFound("doctor not found")
		}
		return nil, httpx.Internal(err)
	}

	specs, err := q.SpecialtiesForDoctor(ctx, id)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	blockRows, err := q.BlocksForDoctor(ctx, id)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	now := s.clock()
	from := now
	to := now.AddDate(0, 0, horizonDays)

	exRows, err := q.ExceptionsForDoctor(ctx, db.ExceptionsForDoctorParams{
		DoctorProfileID: id,
		FromDate:        pgDate(from),
		ToDate:          pgDate(to),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	bookedRows, err := q.BookedSlots(ctx, db.BookedSlotsParams{
		DoctorProfileID: id,
		FromTs:          pgtype.Timestamptz{Time: from, Valid: true},
		ToTs:            pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	posts, err := q.RecentPosts(ctx, id)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	blocks := make([]weeklyschedule.Block, len(blockRows))
	weekly := make([]WeeklyBlock, len(blockRows))
	for i, b := range blockRows {
		start := tod(b.StartTime)
		end := tod(b.EndTime)
		blocks[i] = weeklyschedule.Block{
			DayOfWeek:           int(b.DayOfWeek),
			StartTime:           start,
			EndTime:             end,
			SlotDurationMinutes: int(b.SlotDurationMinutes),
			IsActive:            b.IsActive,
		}
		weekly[i] = WeeklyBlock{
			DayOfWeek:           int(b.DayOfWeek),
			Start:               start.String(),
			End:                 end.String(),
			SlotDurationMinutes: int(b.SlotDurationMinutes),
		}
	}

	exceptions := make([]scheduleexception.Exception, len(exRows))
	for i, e := range exRows {
		exceptions[i] = scheduleexception.Exception{
			Date:     e.ExceptionDate.Time,
			FromTime: tod(e.FromTime),
			ToTime:   tod(e.ToTime),
			Type:     scheduleexception.Type(e.Type),
		}
	}

	booked := map[string][]doctorprofile.TimeOfDay{}
	for _, at := range bookedRows {
		u := at.Time.UTC()
		key := u.Format("2006-01-02")
		booked[key] = append(booked[key], doctorprofile.TimeOfDay{Hour: u.Hour(), Minute: u.Minute()})
	}

	specialties := make([]Specialty, len(specs))
	for i, sp := range specs {
		specialties[i] = Specialty{ID: sp.ID, Name: sp.Name}
	}

	recent := make([]Post, len(posts))
	for i, p := range posts {
		recent[i] = Post{
			ID:          p.ID,
			Title:       p.Title,
			Excerpt:     p.Excerpt,
			Type:        string(p.Type),
			PublishedAt: p.PublishedAt.Time,
		}
	}

	var medLicense *string
	if doc.MedicalLicenseNumber.Valid {
		medLicense = &doc.MedicalLicenseNumber.String
	}

	// ✅ FIX: Resolve Profile Image URL based on Slice 12b spec
	var profileImg *string
	if doc.ProfileImageFileID.Valid {
		u, err := s.files.PresignGet(ctx, doc.ProfileImageFileID.Int64, 86400)
		if err == nil {
			profileImg = &u
		}
	} else if doc.ProfileImageUrl.Valid {
		profileImg = &doc.ProfileImageUrl.String
	}

	resp := &Response{
		ID:                   doc.ID,
		FullName:             doc.FullName,
		Bio:                  doc.Bio,
		ClinicName:           doc.ClinicName,
		ClinicAddress:        doc.ClinicAddress,
		ConsultationFee:      strings.TrimSpace(doc.ConsultationFee),
		Currency:             strings.TrimSpace(doc.Currency),
		AverageRating:        strings.TrimSpace(doc.AverageRating),
		RatingCount:          int(doc.RatingCount),
		MedicalLicenseNumber: medLicense,
		ProfileImageURL:      profileImg, // ✅ Use resolved URL
		Specialties:          specialties,
		WeeklySchedule:       weekly,
		RecentPosts:          recent,
	}

	if slot, ok := availability.NextSlot(from, horizonDays, blocks, exceptions, booked); ok {
		resp.NextAvailableAt = &slot
	}

	return resp, nil
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func tod(t pgtype.Time) doctorprofile.TimeOfDay {
	secs := int(t.Microseconds / 1_000_000)
	return doctorprofile.TimeOfDay{Hour: secs / 3600, Minute: (secs % 3600) / 60}
}
