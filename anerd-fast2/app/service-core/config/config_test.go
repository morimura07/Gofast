package config_test

import (
	"gofast/service-core/config"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//nolint:paralleltest
func TestIsRunningTest(t *testing.T) {
	assert.True(t, config.IsRunningTest())

	// Temporarily modify os.Args to simulate not running in a test environment
	originalArgs := os.Args
	os.Args = []string{"cmd"}
	assert.False(t, config.IsRunningTest())
	os.Args = originalArgs
}

//nolint:paralleltest
func TestMustSetEnv_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	// Temporarily modify os.Args to simulate not running in a test environment
	originalArgs := os.Args
	os.Args = []string{"cmd"}
	defer func() {
		os.Args = originalArgs
	}()

	// This should panic because the required env var is not set and it's not a test run
	config.MustSetEnv(true, "PANIC_TEST_KEY")
}

func TestMustSetEnv(t *testing.T) {
	// Test case 1: Environment variable is set
	t.Setenv("TEST_KEY", "test_value")
	value := config.MustSetEnv(true, "TEST_KEY")
	assert.Equal(t, "test_value", value)

	// Test case 2: Environment variable is not set, but not active
	value = config.MustSetEnv(false, "INACTIVE_KEY")
	assert.Empty(t, value)

	// Test case 3: Environment variable is not set and active, should return "test"
	value = config.MustSetEnv(true, "MISSING_KEY")
	assert.Equal(t, "test", value)
}

func TestLoadConfig(t *testing.T) {
	// Set environment variables for testing
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("PORT", "8080")
	t.Setenv("DOMAIN", "localhost")
	t.Setenv("CORE_URL", "http://localhost:8080")
	t.Setenv("CLIENT_URL", "http://localhost:3001")
	t.Setenv("CRON_TOKEN", "test-token")
	t.Setenv("SERVICE_NAME", "gofast-core")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "testdb")
	t.Setenv("POSTGRES_USER", "testuser")
	t.Setenv("POSTGRES_PASSWORD", "testpassword")

	config := config.LoadConfig()

	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "localhost", config.Domain)
	assert.Equal(t, "http://localhost:8080", config.CoreURL)
	assert.Equal(t, "http://localhost:3001", config.ClientURL)
	assert.Equal(t, "test-token", config.CronToken)
	assert.Equal(t, 5*time.Second, config.ContextTimeout)
	assert.Equal(t, 5*time.Second, config.ReadTimeout)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 120*time.Second, config.IdleTimeout)
	assert.Equal(t, 15*time.Minute, config.AccessTokenExp)
	assert.Equal(t, 30*24*time.Hour, config.RefreshTokenExp)
	assert.Equal(t, int64(10<<20), config.MaxFileSize)
	assert.Equal(t, "localhost", config.PostgresHost)
	assert.Equal(t, "5432", config.PostgresPort)
	assert.Equal(t, "testdb", config.PostgresDB)
	assert.Equal(t, "testuser", config.PostgresUser)
	assert.Equal(t, "testpassword", config.PostgresPassword)
	assert.Equal(t, "gofast-core", config.ServiceName)
}
