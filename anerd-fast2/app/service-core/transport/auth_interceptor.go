package transport

import (
	"context"
	"errors"
	"gofast/pkg"
	"gofast/pkg/auth"
	"gofast/pkg/respond"
	"gofast/service-core/config"
	loginSvc "gofast/service-core/domain/login"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type AuthInterceptor struct {
	cfg       *config.Config
	loginDeps *loginSvc.Deps
}

func NewAuthInterceptor(cfg *config.Config, loginDeps *loginSvc.Deps) *AuthInterceptor {
	return &AuthInterceptor{
		cfg:       cfg,
		loginDeps: loginDeps,
	}
}

// devClaims returns admin claims for dev mode, or nil if not in dev mode.
func devClaims(ctx context.Context, loginDeps *loginSvc.Deps, devUserID string) (*auth.AccessTokenClaims, error) {
	if devUserID == "" {
		return nil, nil //nolint:nilnil // intentional: nil claims means not in dev mode
	}
	slog.WarnContext(ctx, "Bypassing authentication with dev user ID", "user_id", devUserID)
	userID, err := uuid.Parse(devUserID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse dev user ID", "error", err)
		return nil, errors.New("internal server error")
	}

	// Query user from database
	user, err := loginDeps.Store.SelectUserByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to select dev user from database", "error", err)
		return nil, errors.New("internal server error")
	}

	// Check user access (adds plan bits if subscription exists)
	access, err := loginSvc.CheckUserAccess(ctx, loginDeps, user)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check dev user access", "error", err)
		return nil, errors.New("internal server error")
	}

	return &auth.AccessTokenClaims{
		ID:     userID,
		Access: access,
		Avatar: user.Avatar,
		Email:  user.Email,
	}, nil
}

// authenticate validates tokens and returns claims.
func authenticate(ctx context.Context, loginDeps *loginSvc.Deps, accessToken, refreshToken string) (*auth.AccessTokenClaims, *loginSvc.AuthResponse, error) {
	authResponse, err := loginSvc.Refresh(ctx, loginDeps, accessToken, refreshToken)
	if err != nil {
		slog.WarnContext(ctx, "Invalid or expired tokens", "error", err)
		return nil, nil, errors.New("authentication required")
	}

	userID, err := uuid.Parse(authResponse.User.ID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse user ID from auth response", "error", err)
		return nil, nil, errors.New("internal server error")
	}

	claims := &auth.AccessTokenClaims{
		ID:     userID,
		Access: authResponse.User.Access,
		Avatar: authResponse.User.Avatar,
		Email:  authResponse.User.Email,
	}
	return claims, authResponse, nil
}

// tokenCookie creates a cookie with standard auth settings.
func tokenCookie(cfg *config.Config, name, value string) *http.Cookie {
	maxAge := int(cfg.AccessTokenExp)
	if name == "refresh_token" {
		maxAge = int(cfg.RefreshTokenExp)
	}
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Domain:   cfg.Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: cfg.CookieSameSite,
		MaxAge:   maxAge,
	}
}

func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Dev mode bypass
		if claims, err := devClaims(ctx, i.loginDeps, i.cfg.DevUserID); claims != nil {
			ctx = auth.NewContextWithUser(ctx, claims)
			return next(ctx, req)
		} else if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Public routes
		publicRoutes := []string{"/proto.v1.LoginService/LoginURL"}
		for _, route := range publicRoutes {
			if strings.EqualFold(req.Spec().Procedure, route) {
				return next(ctx, req)
			}
		}

		// Get cookies
		httpReq := &http.Request{Header: req.Header()}
		accessCookie, err := httpReq.Cookie("access_token")
		if err != nil {
			slog.WarnContext(ctx, "Missing access token", "error", err)
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		refreshCookie, err := httpReq.Cookie("refresh_token")
		if err != nil {
			slog.WarnContext(ctx, "Missing refresh token", "error", err)
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}

		// Authenticate
		claims, authResponse, err := authenticate(ctx, i.loginDeps, accessCookie.Value, refreshCookie.Value)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		ctx = auth.NewContextWithUser(ctx, claims)

		// Refresh cookies if needed
		if authResponse.Fresh {
			res, err := next(ctx, req)
			if err != nil {
				return nil, err
			}
			res.Header().Add("Set-Cookie", tokenCookie(i.cfg, "access_token", authResponse.AccessToken).String())
			res.Header().Add("Set-Cookie", tokenCookie(i.cfg, "refresh_token", authResponse.RefreshToken).String())
			return res, nil
		}

		return next(ctx, req)
	}
}

func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		// Dev mode bypass
		if claims, err := devClaims(ctx, i.loginDeps, i.cfg.DevUserID); claims != nil {
			ctx = auth.NewContextWithUser(ctx, claims)
			return next(ctx, conn)
		} else if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}

		// Public routes (add streaming routes here if needed)
		publicRoutes := []string{}
		for _, route := range publicRoutes {
			if strings.EqualFold(conn.Spec().Procedure, route) {
				return next(ctx, conn)
			}
		}

		// Get cookies
		httpReq := &http.Request{Header: conn.RequestHeader()}
		accessCookie, err := httpReq.Cookie("access_token")
		if err != nil {
			slog.WarnContext(ctx, "Missing access token", "error", err)
			return connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}
		refreshCookie, err := httpReq.Cookie("refresh_token")
		if err != nil {
			slog.WarnContext(ctx, "Missing refresh token", "error", err)
			return connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
		}

		// Authenticate
		claims, authResponse, err := authenticate(ctx, i.loginDeps, accessCookie.Value, refreshCookie.Value)
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		ctx = auth.NewContextWithUser(ctx, claims)

		// Refresh cookies if needed
		if authResponse.Fresh {
			conn.ResponseHeader().Add("Set-Cookie", tokenCookie(i.cfg, "access_token", authResponse.AccessToken).String())
			conn.ResponseHeader().Add("Set-Cookie", tokenCookie(i.cfg, "refresh_token", authResponse.RefreshToken).String())
		}

		return next(ctx, conn)
	}
}

// WrapStreamingClient is a no-op for a server-side interceptor. It simply
// passes the request to the next interceptor in the chain.
func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// AuthMiddleware wraps an HTTP handler with authentication logic.
func AuthMiddleware(cfg *config.Config, loginDeps *loginSvc.Deps, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Dev mode bypass
		if claims, err := devClaims(ctx, loginDeps, cfg.DevUserID); claims != nil {
			ctx = auth.NewContextWithUser(ctx, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		} else if err != nil {
			respond.JSON(w, r, nil, pkg.InternalError{Err: err})
			return
		}

		// Get cookies
		accessCookie, err := r.Cookie("access_token")
		if err != nil {
			slog.WarnContext(ctx, "Missing access token", "error", err)
			respond.JSON(w, r, nil, pkg.UnauthorizedError{Err: errors.New("authentication required")})
			return
		}
		refreshCookie, err := r.Cookie("refresh_token")
		if err != nil {
			slog.WarnContext(ctx, "Missing refresh token", "error", err)
			respond.JSON(w, r, nil, pkg.UnauthorizedError{Err: errors.New("authentication required")})
			return
		}

		// Authenticate
		claims, authResponse, err := authenticate(ctx, loginDeps, accessCookie.Value, refreshCookie.Value)
		if err != nil {
			respond.JSON(w, r, nil, pkg.UnauthorizedError{Err: err})
			return
		}
		ctx = auth.NewContextWithUser(ctx, claims)

		// Refresh cookies if needed
		if authResponse.Fresh {
			http.SetCookie(w, tokenCookie(cfg, "access_token", authResponse.AccessToken))
			http.SetCookie(w, tokenCookie(cfg, "refresh_token", authResponse.RefreshToken))
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
