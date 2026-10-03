package deleteattachment

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
	attachmentID, err := strconv.ParseInt(chi.URLParam(r, "attachmentId"), 10, 64)
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest("invalid attachment id"))
		return
	}
	if err := h.svc.Execute(r.Context(), u.UserID, attachmentID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}