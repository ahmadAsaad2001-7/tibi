package search

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/availability"
	"tibi/internal/doctors/search/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

const (
	defaultLimit            = 20
	maxLimit                = 100
	availabilityHorizonDays = 30
)

type Service struct {
	db    *database.DB
	clock func() time.Time
}

func NewService(db *database.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

// searchRow is our internal unified row type to avoid duplicating logic for every sqlc generated row.
type searchRow struct {
	ID              int64
	FullName        string
	Bio             string
	ClinicName      string
	ConsultationFee string
	Currency        string
	AverageRating   string
	RatingCount     int
	ProfileImageURL *string
	CreatedAt       time.Time
}

type Item struct {
	ID              int64       `json:"id"`
	FullName        string      `json:"full_name"`
	ClinicName      string      `json:"clinic_name"`
	Bio             string      `json:"bio"`
	ProfileImageURL *string     `json:"profile_image_url"`
	ConsultationFee string      `json:"consultation_fee"`
	Currency        string      `json:"currency"`
	AverageRating   string      `json:"average_rating"`
	RatingCount     int         `json:"rating_count"`
	Specialties     []Specialty `json:"specialties"`
	NextAvailableAt *time.Time  `json:"next_available_at"`
}

type Specialty struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Response struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

func (s *Service) Execute(ctx context.Context, p Params) (*Response, error) {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	cur, err := decodeCursor(p.Cursor)
	if err != nil {
		return nil, httpx.ValidationFailed(map[string]string{"cursor": "invalid"})
	}

	q := db.New(s.db.Querier(ctx))
	fetch := int32(p.Limit + 1)

	// Helpers to convert input types to sqlc/pgtype types
	textPtr := func(str *string) pgtype.Text {
		if str == nil {
			return pgtype.Text{Valid: false}
		}
		return pgtype.Text{String: *str, Valid: true}
	}
	numericPtr := func(str *string) pgtype.Numeric {
		var n pgtype.Numeric
		if str != nil {
			_ = n.Scan(*str)
		}
		return n
	}
	cursorID := pgtype.Int8{Int64: cur.ID, Valid: cur.ID != 0}

	var rows []searchRow

	switch p.Sort {
	case SortRating:
		sqlRows, err := q.SearchDoctorsByRating(ctx, db.SearchDoctorsByRatingParams{
			Q: textPtr(p.Q), SpecialtyIds: p.SpecialtyIDs,
			MinFee: numericPtr(p.MinFee), MaxFee: numericPtr(p.MaxFee), MinRating: numericPtr(p.MinRating),
			CursorRating: numericPtr(cur.Rating), CursorID: cursorID, LimitCount: fetch,
		})
		if err != nil {
			return nil, httpx.Internal(err)
		}
		for _, r := range sqlRows {
			rows = append(rows, searchRow{
				ID: r.ID, FullName: r.FullName, Bio: r.Bio, ClinicName: r.ClinicName,
				ConsultationFee: r.ConsultationFee, Currency: r.Currency,
				AverageRating: r.AverageRating, RatingCount: int(r.RatingCount),
				ProfileImageURL: textToPtr(r.ProfileImageUrl), CreatedAt: r.CreatedAt.Time,
			})
		}

	case SortFeeAsc:
		sqlRows, err := q.SearchDoctorsByFeeAsc(ctx, db.SearchDoctorsByFeeAscParams{
			Q: textPtr(p.Q), SpecialtyIds: p.SpecialtyIDs,
			MinFee: numericPtr(p.MinFee), MaxFee: numericPtr(p.MaxFee), MinRating: numericPtr(p.MinRating),
			CursorFee: numericPtr(cur.Fee), CursorID: cursorID, LimitCount: fetch,
		})
		if err != nil {
			return nil, httpx.Internal(err)
		}
		for _, r := range sqlRows {
			rows = append(rows, searchRow{
				ID: r.ID, FullName: r.FullName, Bio: r.Bio, ClinicName: r.ClinicName,
				ConsultationFee: r.ConsultationFee, Currency: r.Currency,
				AverageRating: r.AverageRating, RatingCount: int(r.RatingCount),
				ProfileImageURL: textToPtr(r.ProfileImageUrl), CreatedAt: r.CreatedAt.Time,
			})
		}

	case SortFeeDesc:
		sqlRows, err := q.SearchDoctorsByFeeDesc(ctx, db.SearchDoctorsByFeeDescParams{
			Q: textPtr(p.Q), SpecialtyIds: p.SpecialtyIDs,
			MinFee: numericPtr(p.MinFee), MaxFee: numericPtr(p.MaxFee), MinRating: numericPtr(p.MinRating),
			CursorFee: numericPtr(cur.Fee), CursorID: cursorID, LimitCount: fetch,
		})
		if err != nil {
			return nil, httpx.Internal(err)
		}
		for _, r := range sqlRows {
			rows = append(rows, searchRow{
				ID: r.ID, FullName: r.FullName, Bio: r.Bio, ClinicName: r.ClinicName,
				ConsultationFee: r.ConsultationFee, Currency: r.Currency,
				AverageRating: r.AverageRating, RatingCount: int(r.RatingCount),
				ProfileImageURL: textToPtr(r.ProfileImageUrl), CreatedAt: r.CreatedAt.Time,
			})
		}

	case SortCreatedAt:
		var cursorTime pgtype.Timestamptz
		if cur.CreatedAt != nil {
			t, _ := time.Parse(time.RFC3339Nano, *cur.CreatedAt)
			cursorTime = pgtype.Timestamptz{Time: t, Valid: true}
		}
		sqlRows, err := q.SearchDoctorsByCreatedAt(ctx, db.SearchDoctorsByCreatedAtParams{
			Q: textPtr(p.Q), SpecialtyIds: p.SpecialtyIDs,
			MinFee: numericPtr(p.MinFee), MaxFee: numericPtr(p.MaxFee), MinRating: numericPtr(p.MinRating),
			CursorCreatedAt: cursorTime, CursorID: cursorID, LimitCount: fetch,
		})
		if err != nil {
			return nil, httpx.Internal(err)
		}
		for _, r := range sqlRows {
			rows = append(rows, searchRow{
				ID: r.ID, FullName: r.FullName, Bio: r.Bio, ClinicName: r.ClinicName,
				ConsultationFee: r.ConsultationFee, Currency: r.Currency,
				AverageRating: r.AverageRating, RatingCount: int(r.RatingCount),
				ProfileImageURL: textToPtr(r.ProfileImageUrl), CreatedAt: r.CreatedAt.Time,
			})
		}

	default:
		return nil, httpx.ValidationFailed(map[string]string{"sort": "invalid"})
	}

	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}

	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}

	specs, err := q.SpecialtiesForDoctors(ctx, ids)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	blocks, err := q.BlocksForDoctors(ctx, ids)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	now := s.clock()
	from := now
	to := now.AddDate(0, 0, availabilityHorizonDays)

	exceptions, err := q.ExceptionsForDoctors(ctx, db.ExceptionsForDoctorsParams{
		DoctorIds: ids,
		FromDate:  pgDate(from),
		ToDate:    pgDate(to),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	booked, err := q.BookedSlotsForDoctors(ctx, db.BookedSlotsForDoctorsParams{
		DoctorIds: ids,
		FromTs:    pgtype.Timestamptz{Time: from, Valid: true},
		ToTs:      pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}

	specsByDoc := groupSpecialties(specs)
	blocksByDoc := groupBlocks(blocks)
	exceptionsByDoc := groupExceptions(exceptions)
	bookedByDoc := groupBooked(booked)

	items := make([]Item, len(rows))
	for i, r := range rows {
		item := Item{
			ID: r.ID, FullName: r.FullName, ClinicName: r.ClinicName, Bio: r.Bio,
			ProfileImageURL: r.ProfileImageURL, ConsultationFee: r.ConsultationFee,
			Currency: r.Currency, AverageRating: r.AverageRating,
			RatingCount: r.RatingCount, Specialties: specsByDoc[r.ID],
		}
		if slot, ok := availability.NextSlot(
			from, availabilityHorizonDays,
			blocksByDoc[r.ID], exceptionsByDoc[r.ID], bookedByDoc[r.ID],
		); ok {
			item.NextAvailableAt = &slot
		}
		items[i] = item
	}

	resp := &Response{Items: items, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		resp.NextCursor = ptr(encodeCursor(cursorFromRow(p.Sort, last)))
	}
	return resp, nil
}

// Helper to convert pgtype.Text to *string
func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
