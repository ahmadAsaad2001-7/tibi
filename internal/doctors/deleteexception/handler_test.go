package deleteexception

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"tibi/internal/platform/auth"
)

func TestDeleteExceptionRejectsBadID(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	route := chi.NewRouteContext()
	route.URLParams.Add("id", "abc")
	ctx := auth.WithUser(context.WithValue(req.Context(), chi.RouteCtxKey, route), auth.AuthUser{UserID: 1, Role: "Doctor"})
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}
