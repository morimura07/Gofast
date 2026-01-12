package auth_test

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"log"
	"maps"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gofast/pkg/auth"
)

var (
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
)

func TestMain(m *testing.M) {
	var err error
	publicKey, privateKey, err = ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatalln("Failed to generate keys:", err)
	}

	privateFile, err := os.CreateTemp("", "private_*.pem")
	if err != nil {
		log.Fatalln("Failed to create temp private key file:", err)
	}
	defer func() {
		if err := os.Remove(privateFile.Name()); err != nil {
			log.Println("Failed to remove temp private key file:", err)
		}
	}()

	publicFile, err := os.CreateTemp("", "public_*.pem")
	if err != nil {
		log.Fatalln("Failed to create temp public key file:", err)
	}
	defer func() {
		if err := os.Remove(publicFile.Name()); err != nil {
			log.Println("Failed to remove temp public key file:", err)
		}
	}()

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalln("Failed to marshal private key:", err)
	}
	if err := pem.Encode(privateFile, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalln("Failed to write private key:", err)
	}
	if err := privateFile.Close(); err != nil {
		log.Fatalln("Failed to close private key file:", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatalln("Failed to marshal public key:", err)
	}
	if err := pem.Encode(publicFile, &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}); err != nil {
		log.Fatalln("Failed to write public key:", err)
	}
	if err := publicFile.Close(); err != nil {
		log.Fatalln("Failed to close public key file:", err)
	}

	if err := os.Setenv("PRIVATE_KEY_PATH", privateFile.Name()); err != nil {
		log.Fatalln("Failed to set PRIVATE_KEY_PATH:", err)
	}
	if err := os.Setenv("PUBLIC_KEY_PATH", publicFile.Name()); err != nil {
		log.Fatalln("Failed to set PUBLIC_KEY_PATH:", err)
	}

	code := m.Run()

	if err := os.Unsetenv("PRIVATE_KEY_PATH"); err != nil {
		log.Println("Failed to unset PRIVATE_KEY_PATH:", err)
	}
	if err := os.Unsetenv("PUBLIC_KEY_PATH"); err != nil {
		log.Println("Failed to unset PUBLIC_KEY_PATH:", err)
	}

	os.Exit(code)
}

func copyMap(m jwt.MapClaims) jwt.MapClaims {
	newMap := make(jwt.MapClaims, len(m))
	maps.Copy(newMap, m)
	return newMap
}

func generateTestToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, claims)
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return tokenString
}

func TestContextUser(t *testing.T) {
	t.Parallel()
	t.Run("should return user from context", func(t *testing.T) {
		t.Parallel()
		claims := &auth.AccessTokenClaims{
			ID:     uuid.New(),
			Access: auth.UserAccess,
		}
		ctx := auth.NewContextWithUser(context.Background(), claims)
		retrievedClaims, ok := auth.UserFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, claims, retrievedClaims)
	})

	t.Run("should return false if no user in context", func(t *testing.T) {
		t.Parallel()
		_, ok := auth.UserFromContext(context.Background())
		assert.False(t, ok)
	})
}

func TestHasAccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		required   int64
		userAccess int64
		expected   bool
	}{
		{"has full access", auth.GetSkeletons | auth.CreateSkeleton, auth.AdminAccess, true},
		{"has exact access", auth.GetSkeletons, auth.GetSkeletons, true},
		{"missing one access", auth.GetSkeletons | auth.CreateSkeleton, auth.GetSkeletons, false},
		{"has no access", auth.GetSkeletons, auth.ProPlan, false},
		{"user has zero access", auth.GetSkeletons, 0, false},
		{"required is zero access", 0, auth.AdminAccess, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			has := auth.HasAccess(tc.required, tc.userAccess)
			assert.Equal(t, tc.expected, has)
		})
	}
}

func TestUpdateAccess(t *testing.T) {
	t.Parallel()

	t.Run("should add access", func(t *testing.T) {
		t.Parallel()
		initialAccess := auth.GetSkeletons
		accessToAdd := auth.CreateSkeleton
		expected := initialAccess | accessToAdd

		updatedAccess := auth.UpdateAccess(initialAccess, accessToAdd)
		assert.Equal(t, expected, updatedAccess)
	})

	t.Run("should not change access if already present", func(t *testing.T) {
		t.Parallel()
		initialAccess := auth.GetSkeletons | auth.CreateSkeleton
		accessToAdd := auth.CreateSkeleton
		expected := initialAccess

		updatedAccess := auth.UpdateAccess(initialAccess, accessToAdd)
		assert.Equal(t, expected, updatedAccess)
	})
}

func TestGenerateAndValidateTokens(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	refreshTokenID := uuid.New().String()
	access := auth.UserAccess
	avatar := "http://example.com/avatar.png"
	email := "test@example.com"

	accessToken, refreshToken, err := auth.GenerateTokens(
		refreshTokenID, userID, access, avatar, email,
		15*time.Minute, 30*24*time.Hour,
	)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	t.Run("validate access token", func(t *testing.T) {
		t.Parallel()
		claims, err := auth.ValidateAccessToken(accessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.ID.String())
		assert.Equal(t, access, claims.Access)
		assert.Equal(t, avatar, claims.Avatar)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("validate access token with bearer prefix", func(t *testing.T) {
		t.Parallel()
		claims, err := auth.ValidateAccessToken("Bearer " + accessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.ID.String())
	})

	t.Run("validate refresh token", func(t *testing.T) {
		t.Parallel()
		claims, err := auth.ValidateRefreshToken(refreshToken)
		require.NoError(t, err)
		assert.Equal(t, refreshTokenID, claims.ID.String())
		assert.Equal(t, userID, claims.UserID.String())
	})
}

func TestGenerateAndValidateSessionToken(t *testing.T) {
	t.Parallel()

	userID := uuid.New().String()
	phone := "+1234567890"

	token, err := auth.GenerateSessionToken(userID, phone, 15*time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	t.Run("validate session token", func(t *testing.T) {
		t.Parallel()
		claims, err := auth.ValidateSessionToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.ID.String())
		assert.Equal(t, phone, claims.Phone)
	})
}

func TestTokenGenerationErrors(t *testing.T) {
	runTests := func(t *testing.T, generate func() error) {
		t.Helper()
		t.Run("no private key path", func(t *testing.T) {
			t.Setenv("PRIVATE_KEY_PATH", "/does/not/exist.pem")
			err := generate()
			assert.Error(t, err)
		})

		t.Run("empty private key", func(t *testing.T) {
			emptyFile, err := os.CreateTemp("", "empty_*.pem")
			require.NoError(t, err)
			defer func() { require.NoError(t, os.Remove(emptyFile.Name())) }()
			require.NoError(t, emptyFile.Close())
			t.Setenv("PRIVATE_KEY_PATH", emptyFile.Name())
			err = generate()
			assert.Error(t, err)
		})

		t.Run("invalid private key", func(t *testing.T) {
			invalidFile, err := os.CreateTemp("", "invalid_*.pem")
			require.NoError(t, err)
			defer func() { require.NoError(t, os.Remove(invalidFile.Name())) }()
			_, err = invalidFile.WriteString("not a pem")
			require.NoError(t, err)
			require.NoError(t, invalidFile.Close())
			t.Setenv("PRIVATE_KEY_PATH", invalidFile.Name())
			err = generate()
			assert.Error(t, err)
		})
	}

	t.Run("GenerateTokens", func(t *testing.T) {
		runTests(t, func() error {
			_, _, err := auth.GenerateTokens(
				uuid.New().String(), uuid.New().String(), 0, "", "",
				15*time.Minute, 30*24*time.Hour,
			)
			return err
		})
	})

	t.Run("GenerateSessionToken", func(t *testing.T) {
		runTests(t, func() error {
			_, err := auth.GenerateSessionToken(uuid.New().String(), "", 15*time.Minute)
			return err
		})
	})
}

func TestTokenValidationErrors(t *testing.T) {
	accessToken, refreshToken, err := auth.GenerateTokens(
		uuid.New().String(), uuid.New().String(), 0, "", "",
		15*time.Minute, 30*24*time.Hour,
	)
	require.NoError(t, err)
	sessionToken, err := auth.GenerateSessionToken(uuid.New().String(), "", 15*time.Minute)
	require.NoError(t, err)

	runTests := func(t *testing.T, validate func(string) error, validToken string) {
		t.Helper()
		t.Run("no public key path", func(t *testing.T) {
			t.Setenv("PUBLIC_KEY_PATH", "/does/not/exist.pem")
			err := validate(validToken)
			assert.Error(t, err)
		})

		t.Run("empty public key", func(t *testing.T) {
			emptyFile, err := os.CreateTemp("", "empty_*.pem")
			require.NoError(t, err)
			defer func() { require.NoError(t, os.Remove(emptyFile.Name())) }()
			require.NoError(t, emptyFile.Close())
			t.Setenv("PUBLIC_KEY_PATH", emptyFile.Name())
			err = validate(validToken)
			assert.Error(t, err)
		})

		t.Run("invalid public key", func(t *testing.T) {
			invalidFile, err := os.CreateTemp("", "invalid_*.pem")
			require.NoError(t, err)
			defer func() { require.NoError(t, os.Remove(invalidFile.Name())) }()
			_, err = invalidFile.WriteString("not a pem")
			require.NoError(t, err)
			require.NoError(t, invalidFile.Close())
			t.Setenv("PUBLIC_KEY_PATH", invalidFile.Name())
			err = validate(validToken)
			assert.Error(t, err)
		})

		t.Run("invalid token string", func(t *testing.T) {
			err := validate("not.a.jwt")
			assert.Error(t, err)
		})

		t.Run("token with invalid signature", func(t *testing.T) {
			_, otherPrivateKey, err := ed25519.GenerateKey(nil)
			require.NoError(t, err)
			token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, jwt.MapClaims{})
			tokenString, err := token.SignedString(otherPrivateKey)
			require.NoError(t, err)

			err = validate(tokenString)
			assert.Error(t, err)
		})

		t.Run("expired token", func(t *testing.T) {
			claims := jwt.MapClaims{
				"id":  uuid.New().String(),
				"exp": time.Now().Add(-time.Hour).Unix(),
			}
			token := generateTestToken(t, claims)
			err := validate(token)
			assert.Error(t, err)
		})
	}

	t.Run("ValidateAccessToken", func(t *testing.T) {
		runTests(t, func(token string) error {
			_, err := auth.ValidateAccessToken(token)
			return err
		}, accessToken)
	})

	t.Run("ValidateRefreshToken", func(t *testing.T) {
		runTests(t, func(token string) error {
			_, err := auth.ValidateRefreshToken(token)
			return err
		}, refreshToken)
	})

	t.Run("ValidateSessionToken", func(t *testing.T) {
		runTests(t, func(token string) error {
			_, err := auth.ValidateSessionToken(token)
			return err
		}, sessionToken)
	})
}

func TestDefaultKeyPaths(t *testing.T) {
	t.Setenv("PRIVATE_KEY_PATH", "")
	t.Setenv("PUBLIC_KEY_PATH", "")

	_, _, err := auth.GenerateTokens(uuid.NewString(), uuid.NewString(), 0, "", "", 15*time.Minute, 30*24*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading private key")

	_, err = auth.ValidateAccessToken("some.test.token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading public key")
}

func TestMalformedClaims(t *testing.T) {
	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, jwt.RegisteredClaims{})
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)

	_, err = auth.ValidateAccessToken(tokenString)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "claims missing 'id' field")
}

func TestAccessTokenInvalidClaims(t *testing.T) {
	t.Parallel()
	baseClaims := jwt.MapClaims{
		"id":     uuid.New().String(),
		"access": float64(auth.UserAccess),
		"avatar": "avatar",
		"email":  "email",
		"exp":    time.Now().Add(time.Hour).Unix(),
	}

	testCases := []struct {
		name          string
		claimModifier func(jwt.MapClaims)
		expectedErr   string
	}{
		{"missing id", func(c jwt.MapClaims) { delete(c, "id") }, "claims missing 'id' field"},
		{"invalid id", func(c jwt.MapClaims) { c["id"] = "not-a-uuid" }, "error parsing user ID"},
		{"non-string id", func(c jwt.MapClaims) { c["id"] = 123 }, "claims missing 'id' field"},
		{"missing access", func(c jwt.MapClaims) { delete(c, "access") }, "claims missing 'access' field"},
		{"non-float64 access", func(c jwt.MapClaims) { c["access"] = "123" }, "claims missing 'access' field"},
		{"missing avatar", func(c jwt.MapClaims) { delete(c, "avatar") }, "claims missing 'avatar' field"},
		{"non-string avatar", func(c jwt.MapClaims) { c["avatar"] = 123 }, "claims missing 'avatar' field"},
		{"missing email", func(c jwt.MapClaims) { delete(c, "email") }, "claims missing 'email' field"},
		{"non-string email", func(c jwt.MapClaims) { c["email"] = 123 }, "claims missing 'email' field"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			claims := copyMap(baseClaims)
			tc.claimModifier(claims)
			token := generateTestToken(t, claims)
			_, err := auth.ValidateAccessToken(token)
			assert.Error(t, err)
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func TestRefreshTokenInvalidClaims(t *testing.T) {
	t.Parallel()
	baseClaims := jwt.MapClaims{
		"id":      uuid.New().String(),
		"user_id": uuid.New().String(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	testCases := []struct {
		name          string
		claimModifier func(jwt.MapClaims)
		expectedErr   string
	}{
		{"missing id", func(c jwt.MapClaims) { delete(c, "id") }, "claims missing 'id' field"},
		{"invalid id", func(c jwt.MapClaims) { c["id"] = "not-a-uuid" }, "error parsing user ID"},
		{"non-string id", func(c jwt.MapClaims) { c["id"] = 123 }, "claims missing 'id' field"},
		{"missing user_id", func(c jwt.MapClaims) { delete(c, "user_id") }, "claims missing 'user_id' field"},
		{"invalid user_id", func(c jwt.MapClaims) { c["user_id"] = "not-a-uuid" }, "error parsing refresh token ID"},
		{"non-string user_id", func(c jwt.MapClaims) { c["user_id"] = 123 }, "claims missing 'user_id' field"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			claims := copyMap(baseClaims)
			tc.claimModifier(claims)
			token := generateTestToken(t, claims)
			_, err := auth.ValidateRefreshToken(token)
			assert.Error(t, err)
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func TestSessionTokenInvalidClaims(t *testing.T) {
	t.Parallel()
	baseClaims := jwt.MapClaims{
		"id":    uuid.New().String(),
		"phone": "+1234567890",
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	testCases := []struct {
		name          string
		claimModifier func(jwt.MapClaims)
		expectedErr   string
	}{
		{"missing id", func(c jwt.MapClaims) { delete(c, "id") }, "claims missing 'id' field"},
		{"invalid id", func(c jwt.MapClaims) { c["id"] = "not-a-uuid" }, "error parsing user ID"},
		{"non-string id", func(c jwt.MapClaims) { c["id"] = 123 }, "claims missing 'id' field"},
		{"missing phone", func(c jwt.MapClaims) { delete(c, "phone") }, "claims missing 'phone' field"},
		{"non-string phone", func(c jwt.MapClaims) { c["phone"] = 12345 }, "claims missing 'phone' field"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			claims := copyMap(baseClaims)
			tc.claimModifier(claims)
			token := generateTestToken(t, claims)
			_, err := auth.ValidateSessionToken(token)
			assert.Error(t, err)
			assert.ErrorContains(t, err, tc.expectedErr)
		})
	}
}
