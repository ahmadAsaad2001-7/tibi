package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ahmadAsaad2001-7/tibi/internal/platform/httpx/trace"
)

// Trace assigns or reuses a trace ID, sets the X-Trace-Id header,
// and stores the ID in the context.
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := trace.FromHeader(r.Header.Get("traceparent"))
		if id == "" {
			id = uuid.NewString()[:32] // 32 hex chars, W3C-shaped
		}
		w.Header().Set("X-Trace-Id", id)
		next.ServeHTTP(w, r.WithContext(trace.WithID(r.Context(), id)))
	})
}

// Recover converts a panic into a 500 and logs it.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("panic",
					"panic", p,
					"path", r.URL.Path,
					"trace_id", trace.ID(r.Context()),
				)
				Error(w, r, Internal(nil))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLog emits one structured line per request.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"trace_id", trace.ID(r.Context()),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(s int) {
	w.status = s
	w.ResponseWriter.WriteHeader(s)
}
