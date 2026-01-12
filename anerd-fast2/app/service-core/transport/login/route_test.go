package login_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	proto "gofast/gen/proto/v1"
	"gofast/pkg/auth"
	pkgtest "gofast/pkg/testutil"
	"gofast/service-core/config"
	"gofast/service-core/domain/login"
	"gofast/service-core/storage/query"
	storetest "gofast/service-core/storage/testutil"
	loginRoute "gofast/service-core/transport/login"

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
	cfg.TwilioServiceSID = ""

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

func TestServer_Refresh(t *testing.T) {
	t.Parallel()
	t.Run("Success - User in context", func(t *testing.T) {
		t.Parallel()
		server := loginRoute.NewLoginServer(login.Deps{}) //nolint:exhaustruct

		user := &auth.AccessTokenClaims{
			ID:     uuid.New(),
			Access: auth.UserAccess,
			Avatar: "",
			Email:  "test@example.com",
		}
		ctx := auth.NewContextWithUser(context.Background(), user)
		req := connect.NewRequest(&proto.RefreshRequest{})

		res, err := server.Refresh(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, user.Email, res.Msg.GetEmail())
		assert.Equal(t, user.Access, res.Msg.GetAccess())
	})

	t.Run("Failure - No user in context", func(t *testing.T) {
		t.Parallel()
		server := loginRoute.NewLoginServer(login.Deps{}) //nolint:exhaustruct

		ctx := context.Background()
		req := connect.NewRequest(&proto.RefreshRequest{})

		_, err := server.Refresh(ctx, req)

		require.Error(t, err)
		connectErr := &connect.Error{}
		require.ErrorAs(t, err, &connectErr)
		assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	})
}

// --- LoginURL Tests ---

func TestServer_LoginURL(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		server := loginRoute.NewLoginServer(env.deps)

		env.mockOAuth.On("AuthCodeURL", mock.Anything, mock.Anything).Return("https://accounts.google.com/oauth", nil).Once()

		req := connect.NewRequest(&proto.LoginRequest{
			Provider:  "google",
			ReturnUrl: "http://localhost:3000",
		})

		res, err := server.LoginURL(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "https://accounts.google.com/oauth", res.Msg.GetUrl())
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Failure - Empty provider", func(t *testing.T) {
		t.Parallel()
		server := loginRoute.NewLoginServer(login.Deps{}) //nolint:exhaustruct

		req := connect.NewRequest(&proto.LoginRequest{
			Provider:  "",
			ReturnUrl: "",
		})

		_, err := server.LoginURL(context.Background(), req)

		require.Error(t, err)
	})

	t.Run("Failure - Unknown provider", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		server := loginRoute.NewLoginServer(env.deps)

		env.mockOAuth.On("AuthCodeURL", mock.Anything, mock.Anything).
			Return("", errors.New("unknown provider: unknown")).Once()

		req := connect.NewRequest(&proto.LoginRequest{
			Provider:  "unknown",
			ReturnUrl: "",
		})

		_, err := server.LoginURL(context.Background(), req)

		require.Error(t, err)
	})
}

// --- LoginCallback Tests ---

func TestServer_LoginCallbackHandler(t *testing.T) {
	t.Parallel()
	t.Run("Success - Existing user", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store)
		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		server := loginRoute.NewLoginServer(env.deps)

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(&login.Info{
				Email:  user.Email,
				Sub:    user.Sub[7:],
				Avatar: user.Avatar,
			}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/login/callback?state="+authToken.ID+"&code=test_code", nil)
		rr := httptest.NewRecorder()

		server.LoginCallback(rr, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
		assert.Equal(t, "http://localhost:3000", rr.Header().Get("Location"))

		cookies := rr.Result().Cookies()
		assert.GreaterOrEqual(t, len(cookies), 2)
		env.mockOAuth.AssertExpectations(t)
	})

	t.Run("Success - New user created", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")

		server := loginRoute.NewLoginServer(env.deps)

		newUserEmail := "newuser-" + uuid.New().String()[:8] + "@test.example.com"

		env.mockOAuth.On("Exchange", mock.Anything, "test_code", mock.Anything).
			Return(&oauth2.Token{AccessToken: "oauth_access_token"}, nil).Once()
		env.mockOAuth.On("GetUserInfo", mock.Anything, "oauth_access_token").
			Return(&login.Info{
				Email:  newUserEmail,
				Sub:    "new_user_sub_123",
				Avatar: "https://example.com/new_avatar.png",
			}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/login/callback?state="+authToken.ID+"&code=test_code", nil)
		rr := httptest.NewRecorder()

		server.LoginCallback(rr, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
		env.mockOAuth.AssertExpectations(t)

		ctx := context.Background()
		users, _ := env.store.SelectAllUsers(ctx)
		found := false
		for _, u := range users {
			if u.Email == newUserEmail {
				found = true
				break
			}
		}
		assert.True(t, found, "new user should be created in DB")
	})

	t.Run("Failure - Missing state/code", func(t *testing.T) {
		t.Parallel()
		server := loginRoute.NewLoginServer(login.Deps{}) //nolint:exhaustruct

		req := httptest.NewRequest(http.MethodGet, "/login/callback", nil)
		rr := httptest.NewRecorder()

		server.LoginCallback(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Failure - Invalid state", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		server := loginRoute.NewLoginServer(env.deps)

		req := httptest.NewRequest(http.MethodGet, "/login/callback?state=nonexistent&code=test_code", nil)
		rr := httptest.NewRecorder()

		server.LoginCallback(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Failure - OAuth exchange error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		authToken := createTestAuthToken(t, env.store, uuid.Nil, "google")
		server := loginRoute.NewLoginServer(env.deps)

		env.mockOAuth.On("Exchange", mock.Anything, "bad_code", mock.Anything).
			Return(nil, assert.AnError).Once()

		req := httptest.NewRequest(http.MethodGet, "/login/callback?state="+authToken.ID+"&code=bad_code", nil)
		rr := httptest.NewRecorder()

		server.LoginCallback(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		env.mockOAuth.AssertExpectations(t)
	})
}

// --- Logout Tests ---

func TestServer_Logout(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		server := loginRoute.NewLoginServer(login.Deps{Cfg: env.cfg}) //nolint:exhaustruct

		req := connect.NewRequest(&proto.LogoutRequest{})
		res, err := server.Logout(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, res)

		cookies := res.Header()["Set-Cookie"]
		require.Len(t, cookies, 2)
		assert.Contains(t, cookies[0], "access_token=;")
		assert.Contains(t, cookies[0], "Max-Age=0")
		assert.Contains(t, cookies[1], "refresh_token=;")
		assert.Contains(t, cookies[1], "Max-Age=0")
	})
}
