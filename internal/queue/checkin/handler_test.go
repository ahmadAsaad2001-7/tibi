package checkin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckInRequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/consultations/1/check-in", nil)
	NewHandler(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}
