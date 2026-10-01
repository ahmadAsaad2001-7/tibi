package listfeed

import (
	"net/http"
	"strconv"

	"tibi/internal/content/doctorpost"
	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"limit": "invalid"}))
			return
		}
		limit = n
	}
	var postType *string
	if raw := q.Get("type"); raw != "" {
		if !doctorpost.PostType(raw).Valid() {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"type": "invalid"}))
			return
		}
		postType = &raw
	}
	var featured *bool
	if raw := q.Get("featured"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"featured": "invalid"}))
			return
		}
		featured = &v
	}
	resp, err := h.svc.Execute(r.Context(), Params{
		Limit:        limit,
		Cursor:       q.Get("cursor"),
		Q:            opt(q.Get("q")),
		Type:         postType,
		FeaturedOnly: featured,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func opt(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
