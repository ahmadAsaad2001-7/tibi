package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/platform/metrics"
	"tibi/internal/platform/trace"
)

const slowQueryThreshold = 100 * time.Millisecond

type traceCtxKey struct{}

type traceState struct {
	start time.Time
	sql   string
	op    string
}

// Tracer logs every query with the trace ID from context. Slow queries
// log at WARN; the rest at DEBUG (SD39). It also updates the DB metrics.
type Tracer struct {
	log *slog.Logger
}

func NewTracer(log *slog.Logger) *Tracer {
	return &Tracer{log: log}
}

func (t *Tracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	state := traceState{
		start: time.Now(),
		sql:   data.SQL,
		op:    opFromSQL(data.SQL),
	}
	return context.WithValue(ctx, traceCtxKey{}, state)
}

func (t *Tracer) TraceQueryEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	state, ok := ctx.Value(traceCtxKey{}).(traceState)
	if !ok {
		return
	}
	dur := time.Since(state.start)

	// Metrics (SD40).
	metrics.DBQueriesTotal.WithLabelValues(state.op).Inc()
	metrics.DBQueryDuration.WithLabelValues(state.op).Observe(dur.Seconds())

	attrs := []any{
		"op", state.op,
		"duration_ms", dur.Milliseconds(),
	}
	if traceID := trace.ID(ctx); traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if data.Err != nil {
		attrs = append(attrs, "err", data.Err.Error())
		t.log.Error("db_query_failed", attrs...)
		return
	}

	if dur >= slowQueryThreshold {
		t.log.Warn("db_query_slow", attrs...)
		return
	}
	t.log.Debug("db_query", attrs...)
}

// opFromSQL extracts the first token as an operation name ("SELECT",
// "INSERT", "UPDATE", ...). Cheap; used as a metric label. Skips leading
// whitespace so multi-line SQL strings (which often begin with a newline) are
// parsed correctly.
func opFromSQL(sql string) string {
	i := 0
	// Skip leading whitespace.
	for i < len(sql) && (sql[i] == ' ' || sql[i] == '\n' || sql[i] == '\t') {
		i++
	}
	start := i
	for i < len(sql) && !(sql[i] == ' ' || sql[i] == '\n' || sql[i] == '\t') {
		i++
	}
	if i-start > 16 {
		i = start + 16
	}
	return sql[start:i]
}