package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIErrorConstructorsAndAs(t *testing.T) {
	details := map[string]string{"email": "required"}
	err := ValidationFailed(details).WithDetails(details)
	he := As(err)
	if he.Status != 400 || he.Code != "validation_failed" || he.Details["email"] != "required" {
		t.Fatalf("validation error = %+v", he)
	}
	if As(BadRequest("x")).Status != 400 || As(Unauthenticated("x")).Status != 401 ||
		As(Forbidden("x")).Status != 403 || As(NotFound("x")).Status != 404 ||
		As(Conflict("x")).Status != 409 || As(AlreadyExists("x")).Code != "already_exists" ||
		As(Unprocessable("x")).Status != 422 {
		t.Fatal("constructor status mismatch")
	}

	cause := errors.New("disk")
	internal := As(Internal(cause))
	if internal.Status != 500 || !errors.Is(internal, cause) {
		t.Fatalf("internal = %+v", internal)
	}
	wrapped := As(errors.New("plain"))
	if wrapped.Code != "internal_error" || wrapped.Err == nil {
		t.Fatalf("plain error was not wrapped: %+v", wrapped)
	}
}

func TestErrorEnvelopeIncludesTrace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req = req.WithContext(WithID(req.Context(), "abc123"))
	rec := httptest.NewRecorder()

	Error(rec, req, NotFound("user not found"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			TraceID string `json:"trace_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "not_found" || body.Error.Message != "user not found" || body.Error.TraceID != "abc123" {
		t.Fatalf("body = %+v", body.Error)
	}
}

func TestTraceID(t *testing.T) {
	if ID(context.Background()) != "" {
		t.Fatal("missing trace id should be empty")
	}
	ctx := WithID(context.Background(), "")
	if ID(ctx) != "" {
		t.Fatal("empty id should not be stored")
	}
	ctx = WithID(context.Background(), "trace")
	if ID(ctx) != "trace" {
		t.Fatalf("id = %q", ID(ctx))
	}

	valid := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	if got := FromHeader(valid); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("FromHeader = %q", got)
	}
	for _, bad := range []string{"", "00-zz-aa-01", "only-one", "00-abc-def-01"} {
		if FromHeader(bad) != "" {
			t.Fatalf("FromHeader(%q) should be empty", bad)
		}
	}
}

func TestTraceAndRecoverMiddleware(t *testing.T) {
	handler := Trace(Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ID(r.Context()) == "" {
			t.Fatal("trace id missing from context")
		}
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("X-Trace-Id") != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace header = %q", rec.Header().Get("X-Trace-Id"))
	}

	fresh := httptest.NewRecorder()
	Trace(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(fresh, httptest.NewRequest(http.MethodGet, "/", nil))
	if len(fresh.Header().Get("X-Trace-Id")) != 32 {
		t.Fatalf("generated trace id = %q", fresh.Header().Get("X-Trace-Id"))
	}
}
