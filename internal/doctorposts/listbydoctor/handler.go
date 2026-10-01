package listbydoctor

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "doctorId"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, httpx.BadRequest("invalid doctor id"))
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"limit": "invalid"}))
			return
		}
		limit = n
	}
	resp, err := h.svc.Execute(r.Context(), Params{
		DoctorProfileID: id,
		Limit:           limit,
		Cursor:          r.URL.Query().Get("cursor"),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
