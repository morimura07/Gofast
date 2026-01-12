package login

import (
	"context"
	"errors"
	"fmt"
	proto "gofast/gen/proto/v1"
	"gofast/pkg"
	"gofast/pkg/auth"
	"gofast/service-core/domain/login"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
)

type Server struct {
	deps login.Deps
}

func NewLoginServer(deps login.Deps) *Server {
	return &Server{deps: deps}
}

func (s *Server) Refresh(
	ctx context.Context,
	_ *connect.Request[proto.RefreshRequest],
) (*connect.Response[proto.RefreshResponse], error) {
	// Claims are already fresh from auth interceptor, which calls loginSvc.Refresh
	// and handles cookie refresh when subscription changes
	claims, ok := auth.UserFromContext(ctx)
	if !ok {
		slog.WarnContext(ctx, "Refresh handler called without user in context")
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	return connect.NewResponse(&proto.RefreshResponse{
		Email:  claims.Email,
		Access: claims.Access,
	}), nil
}

func (s *Server) LoginURL(
	ctx context.Context,
	req *connect.Request[proto.LoginRequest],
) (*connect.Response[proto.LoginResponse], error) {
	if req.Msg.GetProvider() == "" {
		return nil, pkg.BadRequestError{Err: errors.New("provider is empty")}
	}

	url, err := login.Login(ctx, &s.deps, req.Msg.GetReturnUrl(), login.Provider(req.Msg.GetProvider()))
	if err != nil {
		return nil, fmt.Errorf("error getting login url: %w", err)
	}

	return connect.NewResponse(&proto.LoginResponse{Url: url.URL}), nil
}

func (s *Server) LoginCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" || code == "" {
		slog.WarnContext(ctx, "Invalid login callback request: missing state or code")
		http.Error(w, "Invalid request: state and code are required", http.StatusBadRequest)
		return
	}

	authResponse, err := login.Callback(ctx, &s.deps, state, code)
	if err != nil {
		slog.ErrorContext(ctx, "Error in login callback", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    authResponse.AccessToken,
		Domain:   s.deps.Cfg.Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: s.deps.Cfg.CookieSameSite,
		MaxAge:   int(s.deps.Cfg.AccessTokenExp),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    authResponse.RefreshToken,
		Domain:   s.deps.Cfg.Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: s.deps.Cfg.CookieSameSite,
		MaxAge:   int(s.deps.Cfg.RefreshTokenExp),
	})

	http.Redirect(w, r, authResponse.ReturnURL, http.StatusTemporaryRedirect)
}

func (s *Server) Logout(
	_ context.Context,
	_ *connect.Request[proto.LogoutRequest],
) (*connect.Response[proto.LogoutResponse], error) {
	res := connect.NewResponse(&proto.LogoutResponse{})

	accessCookie := http.Cookie{
		Name:     "access_token",
		Value:    "",
		Domain:   s.deps.Cfg.Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: s.deps.Cfg.CookieSameSite,
		MaxAge:   -1,
	}

	refreshCookie := http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Domain:   s.deps.Cfg.Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: s.deps.Cfg.CookieSameSite,
		MaxAge:   -1,
	}

	res.Header().Add("Set-Cookie", accessCookie.String())
	res.Header().Add("Set-Cookie", refreshCookie.String())

	return res, nil
}
