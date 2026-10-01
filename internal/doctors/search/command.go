package search

import "errors"

var ErrInvalidSort = errors.New("invalid sort")

type Sort string

const (
	SortRating    Sort = "rating:desc"
	SortFeeAsc    Sort = "fee:asc"
	SortFeeDesc   Sort = "fee:desc"
	SortCreatedAt Sort = "created_at:desc"
)

func ParseSort(s string) (Sort, error) {
	switch Sort(s) {
	case SortRating, SortFeeAsc, SortFeeDesc, SortCreatedAt:
		return Sort(s), nil
	}
	return "", ErrInvalidSort
}

type Params struct {
	Q            *string
	SpecialtyIDs []int64
	MinFee       *string
	MaxFee       *string
	MinRating    *string
	Sort         Sort
	Limit        int
	Cursor       string
}
