package logger

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	slog.Handler
	serviceAttrs []slog.Attr
	otelLogger   log.Logger
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.HasTraceID() {
		r.AddAttrs(slog.String("trace_id", spanContext.TraceID().String()))
	}
	if spanContext.HasSpanID() {
		r.AddAttrs(slog.String("span_id", spanContext.SpanID().String()))
	}
	if len(h.serviceAttrs) > 0 {
		r.AddAttrs(h.serviceAttrs...)
	}

	// Write to terminal (existing behavior)
	err := h.Handler.Handle(ctx, r)

	// Also send to OTel (new)
	if h.otelLogger != nil {
		var rec log.Record
		rec.SetTimestamp(r.Time)
		rec.SetBody(log.StringValue(r.Message))
		rec.SetSeverity(slogLevelToOTel(r.Level))

		// Add attributes
		r.Attrs(func(a slog.Attr) bool {
			rec.AddAttributes(log.String(a.Key, a.Value.String()))
			return true
		})

		h.otelLogger.Emit(ctx, rec)
	}

	return err
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{
		Handler:      h.Handler.WithAttrs(attrs),
		serviceAttrs: h.serviceAttrs,
		otelLogger:   h.otelLogger,
	}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{
		Handler:      h.Handler.WithGroup(name),
		serviceAttrs: h.serviceAttrs,
		otelLogger:   h.otelLogger,
	}
}

func slogLevelToOTel(level slog.Level) log.Severity {
	switch level {
	case slog.LevelDebug:
		return log.SeverityDebug
	case slog.LevelInfo:
		return log.SeverityInfo
	case slog.LevelWarn:
		return log.SeverityWarn
	case slog.LevelError:
		return log.SeverityError
	default:
		return log.SeverityInfo
	}
}

func InitLogger(logLevel string, serviceName string) {
	var level slog.Level
	if logLevel == "info" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	var handler slog.Handler
	if logLevel == "debug" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	logger := slog.New(&TraceHandler{
		Handler: handler,
		serviceAttrs: []slog.Attr{
			slog.String("service.name", serviceName),
			slog.String("service_name", serviceName),
		},
		otelLogger: global.GetLoggerProvider().Logger(serviceName),
	})

	slog.SetDefault(logger)
}

func Perf(msg string, start time.Time) {
	slog.Info(msg, "duration", time.Since(start))
}
