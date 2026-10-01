package addexception

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tibi/internal/platform/auth"
)

func TestAddExceptionValidation(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"date":"2026-09-30","type":"Nope"}`))
	req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{UserID: 1, Role: "Doctor"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
