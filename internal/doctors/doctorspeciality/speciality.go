package specialty

import "time"

type Specialty struct {
	ID                int64
	Name              string
	ParentSpecialtyID *int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
