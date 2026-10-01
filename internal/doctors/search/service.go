package search

import (
	"context"
	"time"

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

	// Fetch limit+1 to know has_more.
	fetch := int32(p.Limit + 1)
	base := db.SearchBaseParams{
		Q:            p.Q,
		SpecialtyIds: p.SpecialtyIDs,
		MinFee:       p.MinFee,
		MaxFee:       p.MaxFee,
		MinRating:    p.MinRating,
		LimitCount:   fetch,
	}

	var rows []db.SearchRow
	switch p.Sort {
	case SortRating:
		base.CursorRating = cur.Rating
		base.CursorID = curID(cur)
		rows, err = q.SearchDoctorsByRating(ctx, base)
	case SortFeeAsc, SortFeeDesc:
		base.CursorFee = cur.Fee
		base.CursorID = curID(cur)
		if p.Sort == SortFeeAsc {
			rows, err = q.SearchDoctorsByFeeAsc(ctx, base)
		} else {
			rows, err = q.SearchDoctorsByFeeDesc(ctx, base)
		}
	case SortCreatedAt:
		base.CursorCreatedAt = cur.CreatedAt
		base.CursorID = curID(cur)
		rows, err = q.SearchDoctorsByCreatedAt(ctx, base)
	default:
		return nil, httpx.ValidationFailed(map[string]string{"sort": "invalid"})
	}
	if err != nil {
		return nil, httpx.Internal(err)
	}

	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}

	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}

	// Three batch queries regardless of N.
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

	booked, err := q.BookedSlotsForDoctors(ctx, db.BookedSlotsParams{
		DoctorIds: ids,
		FromTs:    from,
		ToTs:      to,
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
			ID:              r.ID,
			FullName:        r.FullName,
			ClinicName:      r.ClinicName,
			Bio:             r.Bio,
			ProfileImageURL: r.ProfileImageUrl,
			ConsultationFee: r.ConsultationFee,
			Currency:        r.Currency,
			AverageRating:   r.AverageRating,
			RatingCount:     int(r.RatingCount),
			Specialties:     specsByDoc[r.ID],
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
	if hasMore {
		last := rows[len(rows)-1]
		resp.NextCursor = ptr(encodeCursor(cursorFromRow(p.Sort, last)))
	}
	return resp, nil
}
