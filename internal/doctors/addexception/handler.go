package addexception

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

var v = validator.New()

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, httpx.Unauthenticated("not authenticated"))
		return
	}
	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		httpx.Error(w, r, httpx.BadRequest("malformed json"))
		return
	}
	if err := v.Struct(cmd); err != nil {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"exception": "invalid exception"}))
		return
	}
	resp, err := h.svc.Execute(r.Context(), u.UserID, cmd)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}
