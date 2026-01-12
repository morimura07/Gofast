package login

import (
	"context"
	"errors"
	"fmt"
	"gofast/pkg"
	"gofast/pkg/auth"
	ot "gofast/pkg/otel"
	"gofast/pkg/str"
	"gofast/service-core/config"
	"gofast/service-core/storage/query"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
)

type Deps struct {
	Cfg    *config.Config
	Store  *query.Queries
	OAuth  OAuthClient
	Twilio TwilioClient
}

type AuthUser struct {
	ID     string `json:"id"`
	Access int64  `json:"access"`
	Avatar string `json:"avatar"`
	Email  string `json:"email"`
}

type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ReturnURL    string    `json:"return_url"`
	User         *AuthUser `json:"user"`
	Fresh        bool      `json:"fresh"`
	// used for 2FA
	SessionToken string `json:"session_token"`
	HasPhone     bool   `json:"has_phone"`
}

type URLResponse struct {
	URL string `json:"url"`
}

func Refresh(ctx context.Context, d *Deps, accessToken string, refreshToken string) (result *AuthResponse, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.Refresh")
	defer func() { done(err) }()

	// Validate token
	claims, err := auth.ValidateAccessToken(accessToken)
	// If access token is invalid, use refresh token flow
	if err != nil {
		return refreshWithToken(ctx, d, span, refreshToken)
	}

	// Access token is valid
	span.AddEvent("Access token is valid")
	user, err := d.Store.SelectUserByID(ctx, claims.ID)
	if err != nil {
		return nil, pkg.NotFoundError{Err: err}
	}
	span.AddEvent("User selected from store")

	access, err := CheckUserAccess(ctx, d, user)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("User access checked")

	// Access changed? Issue fresh tokens
	if claims.Access != access {
		span.AddEvent("Access changed, issuing fresh tokens")
		authResponse, err := createAuthTokens(ctx, d, user)
		if err != nil {
			return nil, pkg.InternalError{Err: err}
		}
		return authResponse, nil
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ReturnURL:    "",
		User: &AuthUser{
			ID:     user.ID.String(),
			Access: access,
			Avatar: user.Avatar,
			Email:  user.Email,
		},
		Fresh:        false,
		HasPhone:     false,
		SessionToken: "",
	}, nil
}

func refreshWithToken(ctx context.Context, d *Deps, span trace.Span, refreshToken string) (*AuthResponse, error) {
	span.AddEvent("Validating refresh token")
	refreshTokenClaims, err := auth.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	// Extract token from database
	refreshTokenStore, err := d.Store.SelectAuthTokenByID(ctx, refreshTokenClaims.ID.String())
	if err != nil {
		return nil, pkg.NotFoundError{Err: err}
	}
	span.AddEvent("Refresh token selected from store")
	// Check if token has a user Id, if not it means the token have been revoked
	if !refreshTokenStore.UserID.Valid {
		return nil, pkg.UnauthorizedError{Err: errors.New("token revoked")}
	}
	// Check if token is expired
	if time.Now().After(refreshTokenStore.Expires) {
		return nil, pkg.UnauthorizedError{Err: errors.New("token expired")}
	}
	// Get user from database
	user, err := d.Store.SelectUserByID(ctx, refreshTokenClaims.UserID)
	if err != nil {
		return nil, pkg.NotFoundError{Err: err}
	}
	span.AddEvent("User selected from store")
	// Create refresh token (valid for 30 days)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	params := query.InsertAuthTokenParams{
		ID:        id.String(),
		Expires:   time.Now().Add(d.Cfg.RefreshTokenExp),
		UserID:    uuid.NullUUID{UUID: refreshTokenClaims.UserID, Valid: true},
		Provider:  "",
		Verifier:  "",
		ReturnUrl: "",
	}
	newRefreshTokenStore, err := d.Store.InsertAuthToken(ctx, params)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("New refresh token inserted into store")
	// Check user access (adds plan bits if subscription exists)
	access, err := CheckUserAccess(ctx, d, user)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("User access checked")
	// Generate JWT tokens
	newAccessToken, newRefreshToken, err := auth.GenerateTokens(
		newRefreshTokenStore.ID,
		user.ID.String(),
		access,
		user.Avatar,
		user.Email,
		d.Cfg.AccessTokenExp,
		d.Cfg.RefreshTokenExp,
	)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("JWT tokens generated")

	// Update user activity and update old refresh token expiration to 1 minute
	// This is to prevent the refresh token from being used again
	go func() {
		ctx, cancel := context.WithTimeout(ctx, d.Cfg.ContextTimeout)
		defer cancel()
		params := query.UpdateAuthTokenParams{
			ID:      refreshTokenClaims.ID.String(),
			Expires: time.Now().Add(time.Minute),
		}
		err := d.Store.UpdateAuthToken(ctx, params)
		if err != nil {
			slog.Error("Error updating token", "error", err)
		}
		err = d.Store.UpdateUserActivity(ctx, user.ID)
		if err != nil {
			slog.Error("Error updating user", "error", err)
		}
	}()

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ReturnURL:    "",
		User: &AuthUser{
			ID:     user.ID.String(),
			Access: access,
			Avatar: user.Avatar,
			Email:  user.Email,
		},
		Fresh:        true,
		SessionToken: "",
		HasPhone:     false,
	}, nil
}

func extractPRIdentifier(d *Deps) string {
	// Extract PR identifier from CLIENT_URL (e.g., "pr-87" from "https://pr-87-context.workers.dev")
	re := regexp.MustCompile(`pr-\d+`)
	match := re.FindString(d.Cfg.ClientURL)
	return match
}

func Login(
	ctx context.Context,
	d *Deps,
	returnURL string,
	provider Provider,
) (result *URLResponse, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.Login")
	defer func() { done(err) }()

	ctx, cancel := context.WithTimeout(ctx, d.Cfg.ContextTimeout)
	defer cancel()

	// Generate random state
	randomState, err := str.GenerateRandomBase64String()
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}

	// Prefix state with PR identifier for PR environments (proxy will route based on this)
	state := randomState
	if prID := extractPRIdentifier(d); prID != "" {
		state = prID + "-" + randomState
		span.AddEvent("Prefixed state with PR identifier: " + prID)
	}

	// Generate code verifier
	verifier := oauth2.GenerateVerifier()
	span.AddEvent("State and verifier generated")

	// Store state and verifier in database
	params := query.InsertAuthTokenParams{
		ID:        state,
		Expires:   time.Now().Add(d.Cfg.AccessTokenExp),
		Provider:  string(provider),
		Verifier:  verifier,
		ReturnUrl: returnURL,
		UserID:    uuid.NullUUID{UUID: uuid.Nil, Valid: false},
	}
	_, err = d.Store.InsertAuthToken(ctx, params)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("Auth token inserted into store")

	// Redirect user to consent page to ask for permission
	url, err := d.OAuth.AuthCodeURL(d.Cfg, provider, state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	if err != nil {
		return nil, pkg.BadRequestError{Err: err}
	}
	return &URLResponse{URL: url}, nil
}

func Callback(
	ctx context.Context,
	d *Deps,
	state string,
	code string,
) (result *AuthResponse, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.Callback")
	defer func() { done(err) }()

	// Get verifier from state
	token, err := d.Store.SelectAuthTokenByID(ctx, state)
	if err != nil {
		return nil, pkg.InternalError{Err: err}
	}
	if time.Now().After(token.Expires) {
		return nil, pkg.InternalError{Err: errors.New("token expired")}
	}
	span.AddEvent("Auth token selected from store")

	// Verify code and exchange for token
	provider := Provider(token.Provider)
	oauthToken, err := d.OAuth.Exchange(ctx, d.Cfg, provider, code, oauth2.VerifierOption(token.Verifier))
	if err != nil {
		return nil, pkg.InternalError{Err: err}
	}
	span.AddEvent("OAuth code exchanged for token")

	// Fetch user info from provider
	userInfo, err := d.OAuth.GetUserInfo(ctx, provider, oauthToken.AccessToken)
	if err != nil {
		return nil, pkg.InternalError{Err: err}
	}
	span.AddEvent("User info fetched from provider")

	// Get user, create if not exists
	userEmail := userInfo.Email
	userSub := fmt.Sprintf("%s:%s", token.Provider, userInfo.Sub)
	userAvatar := userInfo.Avatar
	user, err := d.Store.SelectUserByEmailAndSub(ctx, query.SelectUserByEmailAndSubParams{
		Email: userEmail,
		Sub:   userSub,
	})

	// Create new user if not found
	if err != nil {
		span.AddEvent("User not found, creating new user")
		apiKey, err := str.GenerateRandomHexString()
		if err != nil {
			return nil, pkg.InternalError{Err: err}
		}
		user, err = d.Store.InsertUser(ctx, query.InsertUserParams{
			Email:  userEmail,
			Access: auth.UserAccess,
			Sub:    userSub,
			Avatar: userAvatar,
			ApiKey: apiKey,
		})
		if err != nil {
			return nil, pkg.InternalError{Err: err}
		}
		span.AddEvent("New user inserted into store")
	} else {
		span.AddEvent("User found in store")
	}

	// If Twilio is not configured, skip 2FA
	if d.Cfg.TwilioServiceSID == "" {
		span.AddEvent("Twilio not configured, skipping 2FA")
		authResponse, err := createAuthTokens(ctx, d, user)
		if err != nil {
			return nil, pkg.InternalError{Err: err}
		}
		authResponse.ReturnURL = token.ReturnUrl
		return authResponse, nil
	}

	// If user has a phone number, send a 2FA code
	if user.Phone != "" {
		span.AddEvent("User has phone, sending 2FA code")
		sessionToken, err := Phone(ctx, d, user.ID, user.Phone)
		if err != nil {
			return nil, pkg.InternalError{Err: err}
		}
		return &AuthResponse{
			ReturnURL:    token.ReturnUrl,
			SessionToken: sessionToken,
			HasPhone:     true,
			AccessToken:  "",
			RefreshToken: "",
			User:         nil,
			Fresh:        false,
		}, nil
	}

	// Generate session token
	span.AddEvent("User has no phone, generating session token")
	sessionToken, err := auth.GenerateSessionToken(user.ID.String(), "", d.Cfg.AccessTokenExp)
	if err != nil {
		return nil, pkg.InternalError{Err: err}
	}
	return &AuthResponse{
		ReturnURL:    token.ReturnUrl,
		SessionToken: sessionToken,
		HasPhone:     false,
		AccessToken:  "",
		RefreshToken: "",
		User:         nil,
		Fresh:        false,
	}, nil
}

func Phone(
	ctx context.Context,
	d *Deps,
	userID uuid.UUID,
	phone string,
) (result string, err error) {
	_, span, done := ot.StartSpan(ctx, "login.service.Phone")
	defer func() { done(err) }()

	if phone == "" {
		return "", pkg.UnauthorizedError{Err: errors.New("phone number is required")}
	}

	// Send SMS
	params := &verify.CreateVerificationParams{}
	params.SetTo(phone)
	params.SetChannel("sms")
	_, err = d.Twilio.CreateVerification(d.Cfg.TwilioServiceSID, params)
	if err != nil {
		return "", pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("SMS verification sent")

	// Generate session token
	sessionToken, err := auth.GenerateSessionToken(userID.String(), phone, d.Cfg.AccessTokenExp)
	if err != nil {
		return "", pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("Session token generated")
	return sessionToken, nil
}

func Verify(
	ctx context.Context,
	d *Deps,
	userID uuid.UUID,
	phone string,
	code string,
) (result *AuthResponse, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.Verify")
	defer func() { done(err) }()

	user, err := d.Store.SelectUserByID(ctx, userID)
	if err != nil {
		return nil, pkg.NotFoundError{Err: err}
	}
	span.AddEvent("User selected from store")

	params := &verify.CreateVerificationCheckParams{}
	params.SetTo(phone)
	params.SetCode(code)

	r, err := d.Twilio.CreateVerificationCheck(d.Cfg.TwilioServiceSID, params)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	if *r.Status != "approved" {
		return nil, pkg.UnauthorizedError{Err: errors.New("invalid code")}
	}
	span.AddEvent("SMS verification code verified")

	err = d.Store.UpdateUserPhone(ctx, query.UpdateUserPhoneParams{
		ID:    userID,
		Phone: phone,
	})
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	span.AddEvent("User phone updated in store")

	authResponse, err := createAuthTokens(ctx, d, user)
	if err != nil {
		return nil, pkg.UnauthorizedError{Err: err}
	}
	return authResponse, nil
}

// CheckUserAccess checks the user's subscription status and returns the appropriate access level.
// Plan bits (BasicPlan, ProPlan) are added based on active subscription.

func CheckUserAccess(_ context.Context, _ *Deps, user query.User) (int64, error) {
	return user.Access, nil
}

func ForceRefresh(ctx context.Context, d *Deps, userID uuid.UUID) (result *AuthResponse, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.ForceRefresh")
	defer func() { done(err) }()

	user, err := d.Store.SelectUserByID(ctx, userID)
	if err != nil {
		return nil, pkg.NotFoundError{Err: err}
	}
	span.AddEvent("User selected from store")

	return createAuthTokens(ctx, d, user)
}

func createAuthTokens(ctx context.Context, d *Deps, user query.User) (result *AuthResponse, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.createAuthTokens")
	defer func() { done(err) }()

	// Check user access (adds plan bits if subscription exists)
	access, err := CheckUserAccess(ctx, d, user)
	if err != nil {
		return nil, fmt.Errorf("error checking user access: %w", err)
	}

	// Create refresh token (valid for 30 days)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("error generating UUID: %w", err)
	}
	params := query.InsertAuthTokenParams{
		ID:        id.String(),
		Expires:   time.Now().Add(d.Cfg.RefreshTokenExp),
		UserID:    uuid.NullUUID{UUID: user.ID, Valid: true},
		Provider:  "",
		Verifier:  "",
		ReturnUrl: "",
	}
	refreshToken, err := d.Store.InsertAuthToken(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("error inserting refresh token: %w", err)
	}
	span.AddEvent("Refresh token inserted into store")

	// Generate JWT tokens
	jwtToken, jwtRefreshToken, err := auth.GenerateTokens(
		refreshToken.ID,
		user.ID.String(),
		access,
		user.Avatar,
		user.Email,
		d.Cfg.AccessTokenExp,
		d.Cfg.RefreshTokenExp,
	)
	if err != nil {
		return nil, fmt.Errorf("error generating JWT token: %w", err)
	}
	span.AddEvent("JWT tokens generated")
	return &AuthResponse{
		AccessToken:  jwtToken,
		RefreshToken: jwtRefreshToken,
		ReturnURL:    "",
		User: &AuthUser{
			ID:     user.ID.String(),
			Access: access,
			Avatar: user.Avatar,
			Email:  user.Email,
		},
		Fresh:        true,
		SessionToken: "",
		HasPhone:     false,
	}, nil
}

func HTTPCall(ctx context.Context, url string, accessToken string) (result []byte, err error) {
	ctx, span, done := ot.StartSpan(ctx, "login.service.HTTPCall")
	defer func() { done(err) }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http.DefaultClient.Do: %w", err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			slog.Error("Error closing response body", "error", closeErr)
		}
	}()
	span.AddEvent("HTTP response received")
	userInfoB, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	return userInfoB, nil
}
