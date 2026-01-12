package logger_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"gofast/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput captures stderr from a function.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	require.NoError(t, w.Close())
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestInitLogger(t *testing.T) {
	// Store and restore the default logger
	defaultLogger := slog.Default()
	defer slog.SetDefault(defaultLogger)

	testCases := []struct {
		name              string
		logLevel          string
		logFn             func()
		expectedToContain []string
		mustBeEmpty       bool
	}{
		{
			name:     "info level, log info",
			logLevel: "info",
			logFn: func() {
				slog.Info("info message")
			},
			expectedToContain: []string{"INFO", "info message", "service.name", "gofast-core"},
		},
		{
			name:     "info level, log debug",
			logLevel: "info",
			logFn: func() {
				slog.Debug("debug message")
			},
			mustBeEmpty: true,
		},
		{
			name:     "debug level, log debug",
			logLevel: "debug",
			logFn: func() {
				slog.Debug("debug message")
			},
			expectedToContain: []string{"DEBUG", "debug message"},
		},
		{
			name:     "default level, log debug",
			logLevel: "other",
			logFn: func() {
				slog.Debug("debug message")
			},
			expectedToContain: []string{"DEBUG", "debug message"},
		},
		{
			name:     "log error",
			logLevel: "debug",
			logFn: func() {
				slog.Error("error message")
			},
			expectedToContain: []string{"ERROR", "error message"},
		},
		{
			name:     "log warn",
			logLevel: "debug",
			logFn: func() {
				slog.Warn("warn message")
			},
			expectedToContain: []string{"WARN", "warn message"},
		},
		{
			name:     "unknown log level",
			logLevel: "debug",
			logFn: func() {
				slog.Log(context.Background(), slog.Level(2), "unknown level message")
			},
			expectedToContain: []string{"INFO+2", "unknown level message"},
		},
		{
			name:     "level key with non-level value",
			logLevel: "debug",
			logFn: func() {
				slog.LogAttrs(context.Background(), slog.LevelInfo, "message", slog.String(slog.LevelKey, "not a level"))
			},
			expectedToContain: []string{"INFO", "message", "not a level"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := captureOutput(t, func() {
				logger.InitLogger(tc.logLevel, "gofast-core")
				tc.logFn()
			})

			if tc.mustBeEmpty {
				assert.Empty(t, output)
				return
			}

			for _, expected := range tc.expectedToContain {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestPerf(t *testing.T) {
	// Store and restore the default logger
	defaultLogger := slog.Default()
	defer slog.SetDefault(defaultLogger)

	output := captureOutput(t, func() {
		logger.InitLogger("info", "gofast-core")
		startTime := time.Now()
		time.Sleep(1 * time.Millisecond)
		logger.Perf("performance test", startTime)
	})

	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "performance test")
	assert.Contains(t, output, "duration")
}
