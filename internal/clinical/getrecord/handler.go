package getrecord

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest("invalid consultation id"))
		return
	}
	resp, err := h.svc.Execute(r.Context(), u.UserID, id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}