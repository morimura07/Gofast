package login_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"gofast/pkg/auth"
	pkgtest "gofast/pkg/testutil"
	"gofast/service-core/config"
	"gofast/service-core/domain/login"
	"gofast/service-core/storage/query"
	storetest "gofast/service-core/storage/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	verify "github.com/twilio/twilio-go/rest/verify/v2"
	"golang.org/x/oauth2"
)

// --- Test Setup ---

type testEnv struct {
	db         *sql.DB
	store      *query.Queries
	deps       login.Deps
	cfg        *config.Config
	mockOAuth  *mockOAuthClient
	mockTwilio *mockTwilioClient
	cleanup    func()
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	testDB := pkgtest.SetupTestDB(t)

	cfg := config.LoadConfig()
	cfg.RefreshTokenExp = 30 * 24 * time.Hour
	cfg.ContextTimeout = 5 * time.Second
	cfg.AccessTokenExp = 15 * time.Minute
	cfg.TwilioServiceSID = "" // Disable 2FA by default

	store := query.New(testDB.DB)

	mockOAuth := new(mockOAuthClient)
	mockTwil := new(mockTwilioClient)

	deps := login.Deps{
		Cfg:    cfg,
		Store:  store,
		OAuth:  mockOAuth,
		Twilio: mockTwil,
	}

	return &testEnv{
		db:         testDB.DB,
		store:      store,
		deps:       deps,
		cfg:        cfg,
		mockOAuth:  mockOAuth,
		mockTwilio: mockTwil,
		cleanup:    testDB.Cleanup,
	}
}

// --- Mocks ---

type mockOAuthClient struct {
	mock.Mock
}

func (m *mockOAuthClient) AuthCodeURL(_ *config.Config, _ login.Provider, state string, opts ...oauth2.AuthCodeOption) (string, error) {
	args := m.Called(state, opts)
	return args.String(0), args.Error(1)
}

func (m *mockOAuthClient) Exchange(ctx context.Context, _ *config.Config, _ login.Provider, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	args := m.Called(ctx, code, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}
	token, _ := args.Get(0).(*oauth2.Token)
	return token, args.Error(1) //nolint:wrapcheck
}

func (m *mockOAuthClient) GetUserInfo(ctx context.Context, _ login.Provider, accessToken string) (*login.Info, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}
	info, _ := args.Get(0).(*login.Info)
	return info, args.Error(1) //nolint:wrapcheck
}

type mockTwilioClient struct {
	mock.Mock
}

func (m *mockTwilioClient) CreateVerification(serviceSid string, params *verify.CreateVerificationParams) (*verify.VerifyV2Verification, error) {
	args := m.Called(serviceSid, params)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}
	v, _ := args.Get(0).(*verify.VerifyV2Verification)
	return v, args.Error(1) //nolint:wrapcheck
}

func (m *mockTwilioClient) CreateVerificationCheck(serviceSid string, params *verify.CreateVerificationCheckParams) (*verify.VerifyV2VerificationCheck, error) {
	args := m.Called(serviceSid, params)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck
	}
	vc, _ := args.Get(0).(*verify.VerifyV2VerificationCheck)
	return vc, args.Error(1) //nolint:wrapcheck
}

// --- Test Helpers ---

func createTestAuthToken(t *testing.T, store *query.Queries, userID uuid.UUID, provider string) query.AuthToken {
	t.Helper()
	ctx := context.Background()

	token, err := store.InsertAuthToken(ctx, query.InsertAuthTokenParams{
		ID:        uuid.New().String(),
		Expires:   time.Now().Add(1 * time.Hour),
		UserID:    uuid.NullUUID{UUID: userID, Valid: userID != uuid.Nil},
		Provider:  provider,
		Verifier:  "test_verifier",
		ReturnUrl: "http://localhost:3000",
	})
	require.NoError(t, err)
	return token
}

// --- Refresh Tests ---

func TestService_Refresh(t *testing.T) {
	t.Parallel()
	t.Run("Success - Valid access token returns user without refresh", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		refreshToken := createTestAuthToken(t, env.store, user.ID, "")
		// CheckUserAccess clears plan bits and only adds them back if price ID matches config.
		// Test subscription uses "price_test" which won't match, so no plan bits.
		accessWithoutPlanBits := user.Access &^ auth.BasicPlan &^ auth.ProPlan
		accessToken, refreshTokenJWT, err := auth.GenerateTokens(
			refreshToken.ID,
			user.ID.String(),
			accessWithoutPlanBits,
			user.Avatar,
			user.Email,
			15*time.Minute,
			30*24*time.Hour,
		)
		require.NoError(t, err)

		resp, err := login.Refresh(ctx, &env.deps, accessToken, refreshTokenJWT)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Fresh)
		assert.Equal(t, user.Email, resp.User.Email)
	})

	t.Run("Success - Expired access token triggers refresh", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		refreshToken := createTestAuthToken(t, env.store, user.ID, "")

		_, refreshTokenJWT, err := auth.GenerateTokens(
			refreshToken.ID,
			user.ID.String(),
			user.Access,
			user.Avatar,
			user.Email,
			15*time.Minute,
			30*24*time.Hour,
		)
		require.NoError(t, err)

		resp, err := login.Refresh(ctx, &env.deps, "invalid_access_token", refreshTokenJWT)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Fresh)
		assert.Equal(t, user.Email, resp.User.Email)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("Failure - Invalid refresh token", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		_, err := login.Refresh(ctx, &env.deps, "invalid_access", "invalid_refresh")

		require.Error(t, err)
	})

	t.Run("Failure - Revoked refresh token (no user_id)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		revokedToken := createTestAuthToken(t, env.store, uuid.Nil, "")

		_, refreshTokenJWT, err := auth.GenerateTokens(
			revokedToken.ID,
			user.ID.String(),
			user.Access,
			user.Avatar,
			user.Email,
			15*time.Minute,
			30*24*time.Hour,
		)
		require.NoError(t, err)

		_, err = login.Refresh(ctx, &env.deps, "invalid_access", refreshTokenJWT)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoked")
	})

	t.Run("Failure - Expired refresh token in DB", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		expiredToken, err := env.store.InsertAuthToken(ctx, query.InsertAuthTokenParams{
			ID:        uuid.New().String(),
			Expires:   time.Now().Add(-1 * time.Hour),
			UserID:    uuid.NullUUID{UUID: user.ID, Valid: true},
			Provider:  "",
			Verifier:  "",
			ReturnUrl: "",
		})
		require.NoError(t, err)

		_, refreshTokenJWT, err := auth.GenerateTokens(
			expiredToken.ID,
			user.ID.String(),
			user.Access,
			user.Avatar,
			user.Email,
			15*time.Minute,
			30*24*time.Hour,
		)
		require.NoError(t, err)

		_, err = login.Refresh(ctx, &env.deps, "invalid_access", refreshTokenJWT)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
}

// --- Phone Tests (2FA) ---

func TestService_Phone(t *testing.T) {
	t.Parallel()
	t.Run("Success - SMS sent", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()
		env.cfg.TwilioServiceSID = "test_sid"

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		env.mockTwilio.On("CreateVerification", "test_sid", mock.Anything).
			Return(&verify.VerifyV2Verification{}, nil).Once()

		token, err := login.Phone(ctx, &env.deps, user.ID, "+1234567890")

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		env.mockTwilio.AssertExpectations(t)
	})

	t.Run("Failure - Empty phone", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		_, err := login.Phone(ctx, &env.deps, user.ID, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("Failure - Twilio error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()
		env.cfg.TwilioServiceSID = "test_sid"

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		env.mockTwilio.On("CreateVerification", "test_sid", mock.Anything).
			Return(nil, assert.AnError).Once()

		_, err := login.Phone(ctx, &env.deps, user.ID, "+1234567890")

		require.Error(t, err)
		env.mockTwilio.AssertExpectations(t)
	})
}

// --- Verify Tests (2FA) ---

func TestService_Verify(t *testing.T) {
	t.Parallel()
	t.Run("Success - Code verified", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()
		env.cfg.TwilioServiceSID = "test_sid"

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)
		approvedStatus := "approved"

		env.mockTwilio.On("CreateVerificationCheck", "test_sid", mock.Anything).
			Return(&verify.VerifyV2VerificationCheck{Status: &approvedStatus}, nil).Once()

		resp, err := login.Verify(ctx, &env.deps, user.ID, "+1234567890", "123456")

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		env.mockTwilio.AssertExpectations(t)

		updatedUser, err := env.store.SelectUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "+1234567890", updatedUser.Phone)
	})

	t.Run("Failure - Invalid code", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()
		env.cfg.TwilioServiceSID = "test_sid"

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)
		pendingStatus := "pending"

		env.mockTwilio.On("CreateVerificationCheck", "test_sid", mock.Anything).
			Return(&verify.VerifyV2VerificationCheck{Status: &pendingStatus}, nil).Once()

		_, err := login.Verify(ctx, &env.deps, user.ID, "+1234567890", "wrong_code")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid code")
		env.mockTwilio.AssertExpectations(t)
	})

	t.Run("Failure - User not found", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		_, err := login.Verify(ctx, &env.deps, uuid.New(), "+1234567890", "123456")

		require.Error(t, err)
	})
}

// --- Callback Tests ---

func TestService_Callback(t *testing.T) {
	t.Parallel()
	t.Run("Success - New user created (no 2FA)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		// Create auth token (simulating Login step)
		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(&login.Info{
				Sub:    "google_user_123",
				Email:  "newuser@example.com",
				Avatar: "https://example.com/avatar.png",
			}, nil).Once()

		resp, err := login.Callback(ctx, &env.deps, authToken.ID, "test_code")

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, "http://localhost:3000", resp.ReturnURL)
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Success - Existing user login (no 2FA)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		user := storetest.CreateTestUser(t, env.store)

		// Create auth token
		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		// Extract sub from user (format: "google:uuid")
		userSub := user.Sub[7:] // Remove "google:" prefix

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(&login.Info{
				Sub:    userSub,
				Email:  user.Email,
				Avatar: user.Avatar,
			}, nil).Once()

		resp, err := login.Callback(ctx, &env.deps, authToken.ID, "test_code")

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Success - User with phone triggers 2FA", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()
		env.cfg.TwilioServiceSID = "test_sid"

		ctx := context.Background()

		// Create user with phone
		user, err := env.store.InsertUser(ctx, query.InsertUserParams{
			Email:  "phone-user@example.com",
			Access: auth.UserAccess,
			Sub:    "google:" + uuid.New().String(),
			Avatar: "",
			ApiKey: uuid.New().String(),
		})
		require.NoError(t, err)
		err = env.store.UpdateUserPhone(ctx, query.UpdateUserPhoneParams{
			ID:    user.ID,
			Phone: "+1234567890",
		})
		require.NoError(t, err)

		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")
		userSub := user.Sub[7:]

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(&login.Info{
				Sub:    userSub,
				Email:  user.Email,
				Avatar: "",
			}, nil).Once()
		env.mockTwilio.On("CreateVerification", "test_sid", mock.Anything).
			Return(&verify.VerifyV2Verification{}, nil).Once()

		resp, err := login.Callback(ctx, &env.deps, authToken.ID, "test_code")

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.AccessToken) // No tokens yet - 2FA required
		assert.NotEmpty(t, resp.SessionToken)
		assert.True(t, resp.HasPhone)
		env.mockOAuth.AssertExpectations(t)
		env.mockTwilio.AssertExpectations(t)
	})

	t.Run("Success - New user without phone gets session token (2FA enabled)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()
		env.cfg.TwilioServiceSID = "test_sid"

		ctx := context.Background()
		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(&login.Info{
				Sub:    "new_google_user_456",
				Email:  "nophone@example.com",
				Avatar: "",
			}, nil).Once()

		resp, err := login.Callback(ctx, &env.deps, authToken.ID, "test_code")

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Empty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.SessionToken)
		assert.False(t, resp.HasPhone)
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Failure - Token not found", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		_, err := login.Callback(ctx, &env.deps, "nonexistent_state", "test_code")

		require.Error(t, err)
	})

	t.Run("Failure - Token expired", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		// Create expired token
		expiredToken, err := env.store.InsertAuthToken(ctx, query.InsertAuthTokenParams{
			ID:        uuid.New().String(),
			Expires:   time.Now().Add(-1 * time.Hour),
			UserID:    uuid.NullUUID{UUID: uuid.Nil, Valid: false},
			Provider:  "google",
			Verifier:  "test_verifier",
			ReturnUrl: "http://localhost:3000",
		})
		require.NoError(t, err)

		_, err = login.Callback(ctx, &env.deps, expiredToken.ID, "test_code")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("Failure - Unknown provider", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		// Create token with unknown provider
		unknownProviderToken := createTestAuthToken(t, env.store, uuid.Nil, "unknown")

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(nil, errors.New("unknown provider: unknown")).Once()

		_, err := login.Callback(ctx, &env.deps, unknownProviderToken.ID, "test_code")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})

	t.Run("Failure - Exchange error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		env.mockOAuth.On("Exchange", mock.Anything, "bad_code", mock.Anything).
			Return(nil, assert.AnError).Once()

		_, err := login.Callback(ctx, &env.deps, authToken.ID, "bad_code")

		require.Error(t, err)
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Failure - GetUserInfo error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()
		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(nil, assert.AnError).Once()

		_, err := login.Callback(ctx, &env.deps, authToken.ID, "test_code")

		require.Error(t, err)
		env.mockOAuth.AssertExpectations(t)
	})
}

// --- Login Tests ---

func TestService_Login(t *testing.T) {
	t.Parallel()
	t.Run("Success - Returns OAuth URL", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		env.mockOAuth.On("AuthCodeURL", mock.Anything, mock.Anything).
			Return("https://accounts.google.com/oauth?state=abc", nil).Once()

		resp, err := login.Login(ctx, &env.deps, "http://localhost:3000", login.ProviderGoogle)

		require.NoError(t, err)
		assert.Contains(t, resp.URL, "https://accounts.google.com/oauth")
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Failure - Unknown provider", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background()

		env.mockOAuth.On("AuthCodeURL", mock.Anything, mock.Anything).
			Return("", errors.New("unknown provider: unknown")).Once()

		_, err := login.Login(ctx, &env.deps, "http://localhost:3000", login.Provider("unknown"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}
