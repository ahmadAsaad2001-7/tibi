package refresh

import (
	"encoding/json"
	"net/http"

	"github.com/ahmadAsaad2001-7/tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var cmd Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		httpx.Error(w, r, httpx.BadRequest("malformed json"))
		return
	}
	resp, err := h.svc.Execute(r.Context(), cmd)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
