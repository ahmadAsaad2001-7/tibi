package listspecialties

import (
	"context"

	"tibi/internal/doctors/listspecialties/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct{ db *database.DB }

func NewService(db *database.DB) *Service { return &Service{db: db} }

type Node struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Children []Node `json:"children"`
}

type Response struct {
	Specialties []Node `json:"specialties"`
}

func (s *Service) Execute(ctx context.Context) (*Response, error) {
	q := db.New(s.db.Querier(ctx))
	rows, err := q.ListSpecialties(ctx)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return &Response{Specialties: buildTree(rows)}, nil
}

func buildTree(rows []db.ListSpecialtiesRow) []Node {
	type node struct {
		id       int64
		name     string
		children []*node
	}

	byID := make(map[int64]*node, len(rows))
	for _, r := range rows {
		byID[r.ID] = &node{id: r.ID, name: r.Name}
	}

	roots := make([]*node, 0)
	for _, r := range rows {
		n := byID[r.ID]

		// ✅ FIX 1: التحقق من .Valid بدلاً من مقارنة pgtype.Int8 بـ nil
		if !r.ParentSpecialtyID.Valid {
			roots = append(roots, n)
			continue
		}

		// ✅ FIX 2: استخدام .Int64 لاستخراج القيمة بدلاً من *r.ParentSpecialtyID
		parent, ok := byID[r.ParentSpecialtyID.Int64]
		if !ok {
			continue
		}
		parent.children = append(parent.children, n)
	}

	var toValue func(*node) Node
	toValue = func(n *node) Node {
		out := Node{ID: n.id, Name: n.name, Children: make([]Node, 0, len(n.children))}
		for _, child := range n.children {
			out.Children = append(out.Children, toValue(child))
		}
		return out
	}

	result := make([]Node, 0, len(roots))
	for _, root := range roots {
		result = append(result, toValue(root))
	}
	return result
}
