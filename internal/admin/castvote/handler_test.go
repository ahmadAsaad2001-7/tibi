package castvote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCastVoteRejectsBadRequest(t *testing.T) {
	h := NewHandler(nil)

	badID := httptest.NewRecorder()
	h.ServeHTTP(badID, withParam(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"vote":"For"}`)), "voteId", "abc"))
	if badID.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d", badID.Code)
	}

	badJSON := httptest.NewRecorder()
	h.ServeHTTP(badJSON, withParam(httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{")), "voteId", "4"))
	if badJSON.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d", badJSON.Code)
	}

	badChoice := httptest.NewRecorder()
	h.ServeHTTP(badChoice, withParam(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"vote":"Maybe"}`)), "voteId", "4"))
	if badChoice.Code != http.StatusBadRequest || !strings.Contains(badChoice.Body.String(), "validation_failed") {
		t.Fatalf("bad choice status=%d body=%s", badChoice.Code, badChoice.Body.String())
	}
}

func withParam(r *http.Request, key, value string) *http.Request {
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}
