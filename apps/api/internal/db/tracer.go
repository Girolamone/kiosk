package db

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

// QueryLogger logs every statement the pool runs. It exists to make query
// counts visible during development: an N+1 is invisible in a profiler that
// only shows total time, but obvious when the same statement scrolls past
// twenty times for one request.
type QueryLogger struct {
	logger *slog.Logger
}

func NewQueryLogger(logger *slog.Logger) *QueryLogger {
	return &QueryLogger{logger: logger}
}

func (q *QueryLogger) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	q.logger.Debug("sql", "query", collapse(data.SQL), "args", len(data.Args))
	return ctx
}

func (q *QueryLogger) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// collapse squeezes a multi-line statement onto one line so each query is one
// line of output and counting them is just counting lines.
func collapse(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}
