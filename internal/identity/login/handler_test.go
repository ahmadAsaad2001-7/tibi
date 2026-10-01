package login

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginRejectsMalformedJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("[]"))
	NewHandler(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
