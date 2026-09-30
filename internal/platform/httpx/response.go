package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// Error writes the standard error envelope. Logs the underlying cause.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	he := As(err)
	traceID := ID(r.Context())

	logAttrs := []any{
		"code", he.Code, "status", he.Status,
		"path", r.URL.Path, "method", r.Method,
		"trace_id", traceID,
	}
	if he.Err != nil {
		logAttrs = append(logAttrs, "err", he.Err.Error())
	}
	if he.Status >= 500 {
		slog.Error("http_error", logAttrs...)
	} else {
		slog.Info("http_error", logAttrs...)
	}

	body := map[string]any{
		"error": map[string]any{
			"code":     he.Code,
			"message":  he.Message,
			"details":  he.Details,
			"trace_id": traceID,
		},
	}
	JSON(w, he.Status, body)
}
