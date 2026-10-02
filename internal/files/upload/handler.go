package upload

import (
	"net/http"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		httpx.Error(w, r, httpx.BadRequest("invalid multipart form"))
		return
	}
	f, header, err := r.FormFile("file")
	if err != nil {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"file": "required"}))
		return
	}
	defer f.Close()

	scope := r.FormValue("scope")
	if scope == "" {
		httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"scope": "required"}))
		return
	}

	resp, err := h.svc.Execute(r.Context(), Command{
		UploaderID:  u.UserID,
		Scope:       scope,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Body:        f,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}
