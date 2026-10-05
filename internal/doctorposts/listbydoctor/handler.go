package listbydoctor

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. استخراج doctorId
	id, err := strconv.ParseInt(chi.URLParam(r, "doctorId"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, httpx.BadRequest("invalid doctor id"))
		return
	}

	// 2. استخراج limit (مع تعيين قيمة افتراضية وتحويلها إلى int32)
	limit := int32(20)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"limit": "invalid"}))
			return
		}
		limit = int32(n) // ✅ تحويل إلى int32 ليطابق service.go
	}

	// 3. استخراج cursor وتحويله إلى *int64
	var cursor *int64
	if rawCursor := r.URL.Query().Get("cursor"); rawCursor != "" {
		c, err := strconv.ParseInt(rawCursor, 10, 64)
		if err == nil {
			cursor = &c // ✅ تمرير المؤشر
		}
	}

	// ✅ 4. استدعاء Execute بالوسائط المنفصلة الصحيحة
	resp, err := h.svc.Execute(r.Context(), id, limit, cursor)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}
