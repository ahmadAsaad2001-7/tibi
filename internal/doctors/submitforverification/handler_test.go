package submitforverification

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubmitRequiresAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/profile/doctor/submit-for-verification", nil)
	NewHandler(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}
