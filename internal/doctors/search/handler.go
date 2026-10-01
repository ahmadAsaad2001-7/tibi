package search

import (
	"net/http"
	"strconv"
	"strings"

	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sortRaw := q.Get("sort")
	if sortRaw == "" {
		sortRaw = string(SortRating)
	}
	sort, err := ParseSort(sortRaw)
	if err != nil {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"sort": "invalid"}))
		return
	}

	var specialtyIDs []int64
	if raw := strings.TrimSpace(q.Get("specialty_ids")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || n <= 0 {
				httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"specialty_ids": "invalid"}))
				return
			}
			specialtyIDs = append(specialtyIDs, n)
		}
	}

	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"limit": "invalid"}))
			return
		}
		limit = n
	}

	resp, err := h.svc.Execute(r.Context(), Params{
		Q:            opt(q.Get("q")),
		SpecialtyIDs: specialtyIDs,
		MinFee:       opt(q.Get("min_fee")),
		MaxFee:       opt(q.Get("max_fee")),
		MinRating:    opt(q.Get("min_rating")),
		Sort:         sort,
		Limit:        limit,
		Cursor:       q.Get("cursor"),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if resp.Items == nil {
		resp.Items = []Item{}
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func opt(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
