package proposevote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestProposeRejectsInvalidUserID(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	route := chi.NewRouteContext()
	route.URLParams.Add("userId", "nope")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))

	rec := httptest.NewRecorder()
	h.ServeVerify(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeUnverify(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unverify status = %d", rec.Code)
	}
}
