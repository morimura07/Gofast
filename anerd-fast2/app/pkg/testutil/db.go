package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var (
	initJWTKeysOnce sync.Once
	sharedPool      *pgxpool.Pool
	sharedDB        *sql.DB
	poolOnce        sync.Once
	poolErr         error
)

// TestDB holds database connections for tests
type TestDB struct {
	DB      *sql.DB
	Pool    *pgxpool.Pool
	Cleanup func()
}

// getEnvOrDefault returns the environment variable value or a default.
func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// initSharedPool creates a single shared connection pool for all tests.
func initSharedPool() {
	pgHost := getEnvOrDefault("POSTGRES_HOST", "localhost")
	pgPort := getEnvOrDefault("POSTGRES_PORT", "5432")
	pgUser := getEnvOrDefault("POSTGRES_USER", "postgres")
	pgPassword := getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
	pgDB := getEnvOrDefault("POSTGRES_DB", "postgres")

	host := net.JoinHostPort(pgHost, pgPort)
	// Limit pool size to prevent connection exhaustion during parallel tests
	url := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable&pool_max_conns=10",
		pgUser,
		pgPassword,
		host,
		pgDB,
	)

	sharedPool, poolErr = pgxpool.New(context.Background(), url)
	if poolErr != nil {
		return
	}

	sharedDB = stdlib.OpenDBFromPool(sharedPool)
	if err := sharedDB.PingContext(context.Background()); err != nil {
		sharedPool.Close()
		poolErr = err
	}
}

// SetupTestDB returns a shared database connection for tests.
// Uses a singleton pool to prevent connection exhaustion during parallel tests.
// Fails the test if no database is available.
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// JWT keys for auth service - set via os.Setenv once at init if needed
	initJWTKeysOnce.Do(func() {
		if os.Getenv("PRIVATE_KEY_PATH") == "" {
			_ = os.Setenv("PRIVATE_KEY_PATH", "../../../private.pem")
		}
		if os.Getenv("PUBLIC_KEY_PATH") == "" {
			_ = os.Setenv("PUBLIC_KEY_PATH", "../../../public.pem")
		}
	})

	poolOnce.Do(initSharedPool)

	if poolErr != nil {
		t.Fatalf("no database connection available: %v", poolErr)
	}

	// Cleanup is a no-op since we share the pool across all tests.
	// The pool is closed when the test binary exits.
	return &TestDB{
		DB:      sharedDB,
		Pool:    sharedPool,
		Cleanup: func() {},
	}
}
