package getmetadata

import (
	"context"
	"errors"
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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, httpx.BadRequest("invalid file id"))
		return
	}
	if h.authz != nil {
		if err := h.authz.Authorize(r.Context(), u.UserID, id); err != nil {
			httpx.Error(w, r, err)
			return
		}
	}
	resp, err := h.svc.ExecutePublic(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, r, httpx.NotFound("file not found"))
			return
		}
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
