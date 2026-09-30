package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the subset of the querier surface used by these queries.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db DBTX
}

const listSpecialties = `-- name: ListSpecialties :many
SELECT id, name, parent_specialty_id
FROM doctors_specialties
ORDER BY id
`

type ListSpecialtiesRow struct {
	ID                int64
	Name              string
	ParentSpecialtyID *int64
}

func (q *Queries) ListSpecialties(ctx context.Context) ([]ListSpecialtiesRow, error) {
	rows, err := q.db.Query(ctx, listSpecialties)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ListSpecialtiesRow, 0)
	for rows.Next() {
		var i ListSpecialtiesRow
		if err := rows.Scan(&i.ID, &i.Name, &i.ParentSpecialtyID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
