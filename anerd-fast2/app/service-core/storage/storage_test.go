package storage_test

import (
	"gofast/service-core/config"
	"gofast/service-core/storage"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage(t *testing.T) {
	// These tests require a running Postgres database.
	// Set the required environment variables for the test.
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_DB", "postgres")

	cfg := config.LoadConfig()

	// Test successful connection
	store, clean, err := storage.NewStorage(cfg)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.NotNil(t, clean)
	if clean != nil {
		clean()
	}

	// Test connection failure (ping error)
	cfg.PostgresHost = "invalid-host"
	store, clean, err = storage.NewStorage(cfg)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Nil(t, clean)

	// Test connection failure (invalid port)
	cfg.PostgresPort = "invalid-port"
	store, clean, err = storage.NewStorage(cfg)
	require.Error(t, err)
	assert.Nil(t, store)
	assert.Nil(t, clean)
}
