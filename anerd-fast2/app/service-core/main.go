package main

import (
	"context"
	"gofast/gen/proto/v1/v1connect"
	"gofast/pkg/logger"
	"gofast/pkg/otel"
	"gofast/service-core/config"
	loginSvc "gofast/service-core/domain/login"
	"gofast/service-core/storage"
	"gofast/service-core/storage/query"
	"gofast/service-core/transport"
	loginRoute "gofast/service-core/transport/login"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// GF_MAIN_IMPORT_SERVICES_START
	skeletonSvc "gofast/service-core/domain/skeleton"
	// GF_MAIN_IMPORT_SERVICES_END
	// GF_MAIN_IMPORT_ROUTES_START
	skeletonRoute "gofast/service-core/transport/skeleton"
	// GF_MAIN_IMPORT_ROUTES_END
)

func main() {
	ctx := context.Background()
	// Load the configuration
	cfg := config.LoadConfig()

	// Set up the logger
	logger.InitLogger(cfg.LogLevel, cfg.ServiceName)

	// Set up OpenTelemetry
	shutdown, err := otel.SetupOTelSDK(ctx, cfg.AlloyURL, cfg.ServiceName)
	if err != nil {
		slog.ErrorContext(ctx, "Error setting up OpenTelemetry", "error", err)
		panic(err)
	}
	defer func() {
		err := shutdown(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Error shutting down OpenTelemetry", "error", err)
		}
	}()

	// Connect to the database
	s, clean, err := storage.NewStorage(cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Error opening database", "error", err)
		panic(err)
	}
	defer clean()

	err = s.Conn.PingContext(context.Background())
	if err != nil {
		slog.ErrorContext(ctx, "Error connecting to database", "error", err)
		panic(err)
	}
	slog.InfoContext(ctx, "Database connected")

	// Set up the store
	store := query.New(s.Conn)

	// Initialize login deps
	loginDeps := loginSvc.Deps{
		Cfg:    cfg,
		Store:  store,
		OAuth:  loginSvc.OAuth{},
		Twilio: loginSvc.Twilio{Client: loginSvc.CreateTwilioClient()},
	}

	// Create transport server
	server := transport.NewServer(cfg, store, &loginDeps)

	// Mount login routes
	loginServer := loginRoute.NewLoginServer(loginDeps)
	path, handler := v1connect.NewLoginServiceHandler(loginServer, server.Interceptors())
	server.Mount(path, handler)
	server.MountFunc("/login/callback", loginServer.LoginCallback)

	// GF_MAIN_INIT_SERVICES_START
	skeletonDeps := skeletonSvc.Deps{Store: store}
	// GF_MAIN_INIT_SERVICES_END

	// GF_MAIN_MOUNT_ROUTES_START
	// Mount model routes
	skeletonServer := skeletonRoute.NewSkeletonServer(skeletonDeps)
	path, handler = v1connect.NewSkeletonServiceHandler(skeletonServer, server.Interceptors())
	server.Mount(path, handler)
	// GF_MAIN_MOUNT_ROUTES_END

	// Run the server
	shutdown, errCh := server.Run()

	// Wait for interrupt signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err = <-errCh:
		slog.ErrorContext(ctx, "Server error", "error", err)
		panic(err)
	case sig := <-sigCh:
		slog.InfoContext(ctx, "Received signal, shutting down", "signal", sig)
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = shutdown(shutdownCtx)
	if err != nil {
		slog.ErrorContext(ctx, "Error during shutdown", "error", err)
	}

	slog.InfoContext(ctx, "Server shutdown complete")
}
