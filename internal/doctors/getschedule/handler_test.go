package getschedule

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetScheduleRequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}
