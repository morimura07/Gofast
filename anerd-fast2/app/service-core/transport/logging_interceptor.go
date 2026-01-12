package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
)

type LoggingInterceptor struct{}

func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{}
}

func (i *LoggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		startTime := time.Now()
		var authTokenPresent bool
		httpReq := &http.Request{Header: req.Header()}
		_, err := httpReq.Cookie("access_token")
		if err == nil {
			authTokenPresent = true
		}

		resp, err := next(ctx, req)

		duration := time.Since(startTime)

		logFunc := slog.InfoContext
		if err != nil {
			logFunc = slog.ErrorContext
		}

		logFunc(ctx, "Call",
			slog.String("procedure", req.Spec().Procedure),
			slog.String("duration", duration.String()),
			slog.String("remote_addr", req.Peer().Addr),
			slog.String("user_agent", req.Header().Get("User-Agent")),
			slog.Bool("auth_token_present", authTokenPresent),
			slog.Any("error", err),
		)

		return resp, err
	}
}

func (i *LoggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		startTime := time.Now()
		var authTokenPresent bool
		httpReq := &http.Request{Header: conn.RequestHeader()}
		_, err := httpReq.Cookie("access_token")
		if err == nil {
			authTokenPresent = true
		}
		slog.InfoContext(ctx, "Stream start",
			slog.String("procedure", conn.Spec().Procedure),
			slog.String("remote_addr", conn.Peer().Addr),
			slog.String("user_agent", conn.RequestHeader().Get("User-Agent")),
			slog.Bool("auth_token_present", authTokenPresent),
		)
		err = next(ctx, conn)
		duration := time.Since(startTime)

		logFunc := slog.InfoContext
		if err != nil {
			logFunc = slog.ErrorContext
		}

		logFunc(ctx, "Stream end",
			slog.String("procedure", conn.Spec().Procedure),
			slog.String("duration", duration.String()),
			slog.Any("error", err),
		)
		return err
	}
}

func (i *LoggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}
