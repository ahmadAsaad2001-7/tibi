package webhook

import (
	"io"
	"net/http"

	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ServeHTTP reads the raw body so the HMAC verification can be done over
// the exact bytes Kashier signed. It never decodes the body before verify.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, r, httpx.BadRequest("unable to read body"))
		return
	}

	sig := r.Header.Get("Kashier-Signature")
	if err := h.svc.Execute(r.Context(), rawBody, sig); err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
