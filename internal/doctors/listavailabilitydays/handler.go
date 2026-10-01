package listavailabilitydays

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "doctorProfileId"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, httpx.BadRequest("invalid doctor profile id"))
		return
	}
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"from": "expected YYYY-MM-DD"}))
		return
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"to": "expected YYYY-MM-DD"}))
		return
	}

	resp, err := h.svc.Execute(r.Context(), id, from, to)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
