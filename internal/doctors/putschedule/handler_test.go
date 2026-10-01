package putschedule

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tibi/internal/platform/auth"
)

func TestPutScheduleValidation(t *testing.T) {
	h := NewHandler(nil)
	anon := httptest.NewRecorder()
	h.ServeHTTP(anon, httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"weekly":[]}`)))
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anon status = %d", anon.Code)
	}

	body := `{"weekly":[{"day_of_week":9,"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30,"is_active":true}]}`
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{UserID: 1, Role: "Doctor"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
