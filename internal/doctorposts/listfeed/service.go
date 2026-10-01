package listfeed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"tibi/internal/doctorposts/listfeed/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Params struct {
	Limit        int
	Cursor       string
	Q            *string
	Type         *string
	FeaturedOnly *bool
}

type Specialty struct {
	Name string `json:"name"`
}

type Item struct {
	ID                    int64       `json:"id"`
	Title                 string      `json:"title"`
	Excerpt               string      `json:"excerpt"`
	Type                  string      `json:"type"`
	CoverImageURL         *string     `json:"cover_image_url"`
	ViewCount             int         `json:"view_count"`
	LikeCount             int         `json:"like_count"`
	PublishedAt           time.Time   `json:"published_at"`
	DoctorID              int64       `json:"doctor_id"`
	DoctorFullName        string      `json:"doctor_full_name"`
	DoctorProfileImageURL *string     `json:"doctor_profile_image_url"`
	Specialties           []Specialty `json:"specialties"`
}

type Response struct {
	Items      []Item  `json:"items"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type cursor struct {
	PublishedAt time.Time `json:"p"`
	ID          int64     `json:"i"`
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
	var curAt *time.Time
	var curID *int64
	if cur != nil {
		curAt = &cur.PublishedAt
		curID = &cur.ID
	}

	q := db.New(s.db.Querier(ctx))
	rows, err := q.ListFeed(ctx, db.ListParams{
		Type:              p.Type,
		Q:                 p.Q,
		FeaturedOnly:      p.FeaturedOnly,
		CursorPublishedAt: curAt,
		CursorID:          curID,
		LimitCount:        int32(p.Limit + 1),
	})
	if err != nil {
		return nil, httpx.Internal(err)
	}
	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}
	items, err := s.attach(ctx, q, rows)
	if err != nil {
		return nil, err
	}
	resp := &Response{Items: items, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		encoded := encodeCursor(cursor{PublishedAt: last.PublishedAt, ID: last.ID})
		resp.NextCursor = &encoded
	}
	return resp, nil
}

func (s *Service) attach(ctx context.Context, q *db.Queries, rows []db.PostRow) ([]Item, error) {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.DoctorID
	}
	specs, err := q.Specialties(ctx, ids)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	byDoc := map[int64][]Specialty{}
	for _, sp := range specs {
		byDoc[sp.DoctorProfileID] = append(byDoc[sp.DoctorProfileID], Specialty{Name: sp.Name})
	}
	items := make([]Item, len(rows))
	for i, r := range rows {
		sp := byDoc[r.DoctorID]
		if sp == nil {
			sp = []Specialty{}
		}
		items[i] = Item{
			ID:                    r.ID,
			Title:                 r.Title,
			Excerpt:               r.Excerpt,
			Type:                  r.Type,
			CoverImageURL:         r.CoverImageURL,
			ViewCount:             int(r.ViewCount),
			LikeCount:             int(r.LikeCount),
			PublishedAt:           r.PublishedAt,
			DoctorID:              r.DoctorID,
			DoctorFullName:        r.DoctorFullName,
			DoctorProfileImageURL: r.DoctorProfileImageURL,
			Specialties:           sp,
		}
	}
	return items, nil
}

func decodeCursor(s string) (*cursor, error) {
	if s == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}
