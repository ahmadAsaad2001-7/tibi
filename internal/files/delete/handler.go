package delete

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type FileAccessChecker interface {
	Authorize(ctx context.Context, userID, fileID int64) error
}

type Handler struct {
	svc   *Service
	authz FileAccessChecker
}

func NewHandler(svc *Service, authz FileAccessChecker) *Handler {
	return &Handler{svc: svc, authz: authz}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, httpx.Unauthenticated("not authenticated"))
		return
	}
	fileID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || fileID <= 0 {
		httpx.Error(w, r, httpx.BadRequest("invalid file id"))
		return
	}
	if h.authz != nil {
		if err := h.authz.Authorize(r.Context(), u.UserID, fileID); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	if err := h.svc.Execute(r.Context(), fileID, u.UserID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
