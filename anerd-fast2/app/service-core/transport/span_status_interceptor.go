package transport

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type SpanStatusInterceptor struct{}

// NewSpanStatusInterceptor creates an interceptor that ensures span status is set correctly
// for both unary and streaming RPCs.
// The otelconnect library sets span status to Unset for successful requests,
// but we want explicit Ok status for success.
func NewSpanStatusInterceptor() *SpanStatusInterceptor {
	return &SpanStatusInterceptor{}
}

func (i *SpanStatusInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)

		// Get the current span (created by otelconnect)
		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			return resp, err
		}

		// Set span status based on the result
		if err == nil {
			// Success - explicitly set to Ok
			span.SetStatus(codes.Ok, "")
		} else {
			// Error - ensure error status is set
			// Note: otelconnect may have already set this, but we ensure it's set
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				// For connect errors, use the error message
				span.SetStatus(codes.Error, connectErr.Message())
			} else {
				// For other errors
				span.SetStatus(codes.Error, err.Error())
			}
		}

		return resp, err
	}
}

func (i *SpanStatusInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)

		// Get the current span (created by otelconnect)
		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			return err
		}

		// Set span status based on the result
		if err == nil {
			// Success - explicitly set to Ok
			span.SetStatus(codes.Ok, "")
		} else {
			// Error - ensure error status is set
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				span.SetStatus(codes.Error, connectErr.Message())
			} else {
				span.SetStatus(codes.Error, err.Error())
			}
		}

		return err
	}
}

// WrapStreamingClient is a no-op for a server-side interceptor. It simply
// passes the request to the next interceptor in the chain.
func (i *SpanStatusInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}
