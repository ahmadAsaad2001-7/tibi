package listnotifications

import (
	"net/http"
	"strconv"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, httpx.Unauthenticated("not authenticated"))
		return
	}

	q := r.URL.Query()
	page := 1
	if raw := q.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err == nil {
			page = n
		}
	}
	perPage := 50
	if raw := q.Get("per_page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err == nil {
			perPage = n
		}
	}

	resp, err := h.svc.Execute(r.Context(), u.UserID, page, perPage)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
