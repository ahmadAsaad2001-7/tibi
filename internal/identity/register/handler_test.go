package register

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterRejectsBadInput(t *testing.T) {
	h := NewHandler(nil)

	malformed := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{"))
	h.ServeHTTP(malformed, req)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed status = %d", malformed.Code)
	}

	invalid := httptest.NewRecorder()
	body := `{"email":"not-an-email","password":"short","role":"Admin","full_name":"A","phone_number":"123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	h.ServeHTTP(invalid, req)
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "validation_failed") {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}
