package patientdoctorprotocols

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, httpx.Unauthenticated("not authenticated"))
		return
	}

	patientID, err := strconv.ParseInt(chi.URLParam(r, "patientId"), 10, 64)
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest("invalid patient id"))
		return
	}

	resp, err := h.svc.Execute(r.Context(), u.UserID, patientID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
