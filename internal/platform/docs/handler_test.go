package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpecServesOpenAPI(t *testing.T) {
	rec := httptest.NewRecorder()
	Spec(rec, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "openapi: 3.1.0") {
		t.Fatalf("missing openapi version: %s", body[:min(80, len(body))])
	}
	if !strings.Contains(body, "Tibi API") {
		t.Fatal("missing title")
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "yaml") {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestUIServesScalar(t *testing.T) {
	rec := httptest.NewRecorder()
	UI(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "@scalar/api-reference") {
		t.Fatal("missing scalar script")
	}
	if !strings.Contains(body, "/openapi.yaml") {
		t.Fatal("missing openapi url")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
