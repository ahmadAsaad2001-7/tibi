package proposevote

import (
	"net/http"
	"strconv" // <-- 1. Add this import

	"github.com/go-chi/chi/v5"

	"tibi/internal/admin/adminvote"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeVerify(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, adminvote.ActionVerifyDoctor)
}

func (h *Handler) ServeUnverify(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, adminvote.ActionUnverifyDoctor)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request, action adminvote.ActionType) {
	u, _ := auth.UserFromContext(r.Context())

	// 2. Get the string parameter
	userIDStr := chi.URLParam(r, "userId")

	// 3. Parse it using strconv.ParseInt (returns int64, error)
	targetID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest("invalid user id"))
		return
	}

	resp, err := h.svc.Execute(r.Context(), Input{
		Action:       action,
		AdminUserID:  u.UserID,
		TargetUserID: targetID,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}
