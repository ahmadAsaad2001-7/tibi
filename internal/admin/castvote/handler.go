package castvote

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tibi/internal/admin/adminvote"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type request struct {
	Vote string `json:"vote" validate:"required,oneof=For Against"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	voteID, err := strconv.ParseInt(chi.URLParam(r, "voteId"), 10, 64)
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest("invalid vote id"))
		return
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, r, httpx.BadRequest("malformed json"))
		return
	}
	if req.Vote != "For" && req.Vote != "Against" {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{
			"vote": "must be For or Against",
		}))
		return
	}

	resp, err := h.svc.Execute(r.Context(), Command{
		VoteID:      voteID,
		AdminUserID: u.UserID,
		Choice:      adminvote.Choice(req.Vote),
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
