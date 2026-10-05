package listfeed

import (
	"net/http"
	"strconv"

	"tibi/internal/platform/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// 1. استخراج limit (مع تعيين قيمة افتراضية 20)
	limit := int32(20)
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			httpx.Error(w, r, httpx.ValidationFailed(map[string]string{"limit": "invalid"}))
			return
		}
		limit = int32(n)
	}

	// 2. استخراج cursor وتحويله إلى *int64
	var cursor *int64
	if raw := q.Get("cursor"); raw != "" {
		c, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			cursor = &c
		}
	}

	// ✅ 3. استدعاء Execute بالوسائط المنفصلة الصحيحة
	resp, err := h.svc.Execute(r.Context(), limit, cursor)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}
