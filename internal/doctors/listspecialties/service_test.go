package listspecialties

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"tibi/internal/doctors/listspecialties/db"
)

func TestBuildTreeNestsChildren(t *testing.T) {
	rows := []db.ListSpecialtiesRow{
		{ID: 1, Name: "Cardiology"},
		{ID: 2, Name: "Interventional Cardiology", ParentSpecialtyID: pgtype.Int8{Int64: 1, Valid: true}},
		{ID: 3, Name: "Pediatric Cardiology", ParentSpecialtyID: pgtype.Int8{Int64: 2, Valid: true}},
		{ID: 9, Name: "Orphan", ParentSpecialtyID: pgtype.Int8{Int64: 99, Valid: true}},
	}

	roots := buildTree(rows)
	if len(roots) != 1 || roots[0].Name != "Cardiology" {
		t.Fatalf("roots = %+v", roots)
	}
	if len(roots[0].Children) != 1 || roots[0].Children[0].Name != "Interventional Cardiology" {
		t.Fatalf("children = %+v", roots[0].Children)
	}
	grand := roots[0].Children[0].Children
	if len(grand) != 1 || grand[0].ID != 3 {
		t.Fatalf("grandchildren = %+v", grand)
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	roots := buildTree(nil)
	if roots == nil || len(roots) != 0 {
		t.Fatalf("roots = %#v, want empty slice", roots)
	}
}
