package updateprofile

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tibi/internal/platform/auth"
)

func TestUpdateProfileRequestValidation(t *testing.T) {
	h := NewHandler(nil)

	anon := httptest.NewRecorder()
	h.ServeHTTP(anon, httptest.NewRequest(http.MethodPatch, "/api/v1/profile/doctor", strings.NewReader(`{}`)))
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anon status = %d", anon.Code)
	}

	authed := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/profile/doctor", strings.NewReader(body))
		req = req.WithContext(auth.WithUser(req.Context(), auth.AuthUser{UserID: 1, Role: "PendingDoctor"}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := authed("{"); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed status = %d", rec.Code)
	}
	if rec := authed(`{"unknown":true}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := authed(`{"consultation_fee":"-5","currency":"egp"}`); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "validation_failed") {
		t.Fatalf("invalid fee/currency status=%d body=%s", rec.Code, rec.Body.String())
	}
}
