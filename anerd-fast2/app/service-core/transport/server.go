package transport

import (
	"context"
	"errors"
	"gofast/pkg"
	"gofast/pkg/respond"
	"gofast/service-core/config"
	loginSvc "gofast/service-core/domain/login"
	"gofast/service-core/storage/query"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/otelconnect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	cfg          *config.Config
	store        *query.Queries
	mux          *http.ServeMux
	interceptors connect.Option
}

func NewServer(
	cfg *config.Config,
	store *query.Queries,
	loginDeps *loginSvc.Deps,
) *Server {
	mux := http.NewServeMux()

	// Set up interceptors
	authInterceptor := NewAuthInterceptor(cfg, loginDeps)
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		slog.Error("Error creating OpenTelemetry interceptor", "error", err)
		panic(err)
	}
	interceptors := connect.WithInterceptors(
		otelInterceptor,
		NewSpanStatusInterceptor(),
		NewErrorInterceptor(),
		NewLoggingInterceptor(),
		authInterceptor,
	)

	s := &Server{
		cfg:          cfg,
		store:        store,
		mux:          mux,
		interceptors: interceptors,
	}

	// Register health checks and cron routes
	s.registerHealthChecks()
	s.registerCronRoutes()

	return s
}

func (s *Server) Interceptors() connect.Option { //nolint:ireturn
	return s.interceptors
}

func (s *Server) Mount(path string, handler http.Handler) {
	s.mux.Handle(path, s.withCORS(handler))
}

func (s *Server) MountFunc(pattern string, handler http.HandlerFunc) {
	s.mux.Handle(pattern, s.withCORS(handler))
}

func (s *Server) registerCronRoutes() {
	cronMux := http.NewServeMux()
	cronMux.HandleFunc("POST /crons", func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "Received cron ping")
		respond.JSON(w, r, "pong", nil)
	})
	s.mux.Handle("/crons/", s.cronAuthMiddleware(cronMux))
}

func (s *Server) Run() (shutdown func(context.Context) error, errCh <-chan error) {
	errChannel := make(chan error, 1)
	server := &http.Server{
		Addr:              ":" + s.cfg.Port,
		Handler:           h2c.NewHandler(s.mux, &http2.Server{}),
		ReadHeaderTimeout: s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}

	go func() {
		slog.Info("Server listening on", "port", s.cfg.Port)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errChannel <- err
		}
		close(errChannel)
	}()

	return server.Shutdown, errChannel
}

func (s *Server) registerHealthChecks() {
	ctx := context.Background()
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			panic(err)
		}
	})
	s.mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) {
		_, err := s.store.Ping(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Error pinging database", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			panic(err)
		}
	})
}

func (s *Server) withCORS(h http.Handler) http.Handler {
	//nolint: exhaustruct
	middleware := cors.New(cors.Options{
		AllowedOrigins:   []string{s.cfg.ClientURL},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		AllowCredentials: true,
		ExposedHeaders:   connectcors.ExposedHeaders(),
	})
	return middleware.Handler(h)
}

func (s *Server) cronAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respond.JSON(w, r, nil, pkg.UnauthorizedError{Err: errors.New("missing authorization header")})
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || strings.ToLower(tokenParts[0]) != "bearer" {
			respond.JSON(w, r, nil, pkg.UnauthorizedError{Err: errors.New("invalid authorization header format")})
			return
		}

		if tokenParts[1] != s.cfg.CronToken {
			respond.JSON(w, r, nil, pkg.UnauthorizedError{Err: errors.New("invalid cron token")})
			return
		}

		next.ServeHTTP(w, r)
	})
}
