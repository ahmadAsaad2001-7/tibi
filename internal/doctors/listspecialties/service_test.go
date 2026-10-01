package listspecialties

import (
	"testing"

	"tibi/internal/doctors/listspecialties/db"
)

func TestBuildTreeNestsChildren(t *testing.T) {
	parent := int64(1)
	child := int64(2)
	rows := []db.ListSpecialtiesRow{
		{ID: 1, Name: "Cardiology"},
		{ID: 2, Name: "Interventional Cardiology", ParentSpecialtyID: &parent},
		{ID: 3, Name: "Pediatric Cardiology", ParentSpecialtyID: &child},
		{ID: 9, Name: "Orphan", ParentSpecialtyID: int64Ptr(99)},
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

func int64Ptr(v int64) *int64 { return &v }
