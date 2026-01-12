package login_test

import (
	"context"
	"testing"

	"gofast/service-core/config"
	"gofast/service-core/domain/login"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOAuth_AuthCodeURL(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig()
	cfg.GoogleClientID = "google-id"
	cfg.GoogleClientSecret = "google-secret"
	cfg.OAuthRedirectURI = "http://localhost/callback"
	oauth := login.OAuth{}

	t.Run("Success - Google", func(t *testing.T) {
		t.Parallel()
		url, err := oauth.AuthCodeURL(cfg, login.ProviderGoogle, "test-state")
		require.NoError(t, err)
		assert.Contains(t, url, "accounts.google.com")
		assert.Contains(t, url, "state=test-state")
	})

	t.Run("Success - Github", func(t *testing.T) {
		t.Parallel()
		cfg.GithubClientID = "github-id"
		cfg.GithubClientSecret = "github-secret"
		url, err := oauth.AuthCodeURL(cfg, login.ProviderGithub, "test-state")
		require.NoError(t, err)
		assert.Contains(t, url, "github.com")
	})

	t.Run("Failure - Unknown provider", func(t *testing.T) {
		t.Parallel()
		_, err := oauth.AuthCodeURL(cfg, login.Provider("unknown"), "test-state")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}

func TestOAuth_Exchange(t *testing.T) {
	t.Parallel()

	cfg := config.LoadConfig()
	cfg.GoogleClientID = "google-id"
	cfg.GoogleClientSecret = "google-secret"
	cfg.OAuthRedirectURI = "http://localhost/callback"
	oauth := login.OAuth{}

	t.Run("Failure - Unknown provider", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		_, err := oauth.Exchange(ctx, cfg, login.Provider("unknown"), "code", oauth2.VerifierOption("verifier"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}

func TestOAuth_GetUserInfo(t *testing.T) {
	t.Parallel()

	oauth := login.OAuth{}

	t.Run("Failure - Unknown provider", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		_, err := oauth.GetUserInfo(ctx, login.Provider("unknown"), "token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}
