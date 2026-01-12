package str_test

import (
	"encoding/base64"
	"errors"
	"gofast/pkg/str"
	"io"
	"sync"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	// Test case 1: Valid RFC3339 date string
	rfc3339Str := "2023-10-27T10:00:00Z"
	expected, _ := time.Parse(time.RFC3339, rfc3339Str)
	result, err := str.ParseDate(rfc3339Str)
	if err != nil {
		t.Errorf("unexpected error for input %s: %v", rfc3339Str, err)
	}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	// Test case 2: Valid YYYY-MM-DD date string
	yyyyMMDDStr := "2023-10-27"
	expected, _ = time.Parse("2006-01-02", yyyyMMDDStr)
	result, err = str.ParseDate(yyyyMMDDStr)
	if err != nil {
		t.Errorf("unexpected error for input %s: %v", yyyyMMDDStr, err)
	}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	// Test case 3: Invalid date string
	invalidStr := "27-10-2023"
	_, err = str.ParseDate(invalidStr)
	if err == nil {
		t.Error("expected an error for invalid date string, but got none")
	}

	// Test case 4: Empty string
	emptyStr := ""
	_, err = str.ParseDate(emptyStr)
	if err == nil {
		t.Error("expected an error for empty string, but got none")
	}
}

func TestParseInt32(t *testing.T) {
	t.Parallel()
	// Test case 1: Valid integer string
	intStr := "12345"
	expected := int32(12345)
	result, err := str.ParseInt32(intStr)
	if err != nil {
		t.Errorf("unexpected error for input %s: %v", intStr, err)
	}
	if result != expected {
		t.Errorf("expected %d, got %d", expected, result)
	}

	// Test case 2: Empty string
	emptyStr := ""
	result, err = str.ParseInt32(emptyStr)
	if err != nil {
		t.Errorf("unexpected error for empty string: %v", err)
	}
	if result != 0 {
		t.Errorf("expected 0 for empty string, got %d", result)
	}

	// Test case 3: Invalid integer string
	invalidStr := "abc"
	_, err = str.ParseInt32(invalidStr)
	if err == nil {
		t.Error("expected an error for invalid integer string, but got none")
	}
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock reader error")
}

var _ io.Reader = (*errorReader)(nil)

var randReadLock sync.Mutex

func TestGenerateRandomString(t *testing.T) {
	t.Parallel()

	t.Run("base64", func(t *testing.T) {
		t.Parallel()
		randReadLock.Lock()
		defer randReadLock.Unlock()

		length := 32
		state, err := str.GenerateRandomBase64String()
		if err != nil {
			t.Errorf("error generating random state: %v", err)
		}
		expectedLength := base64.StdEncoding.EncodedLen(length)
		if len(state) != expectedLength {
			t.Errorf("expected state length of %d, but got %d", expectedLength, len(state))
		}
	})

	t.Run("hex", func(t *testing.T) {
		t.Parallel()
		randReadLock.Lock()
		defer randReadLock.Unlock()

		length := 32
		randomString, err := str.GenerateRandomHexString()
		if err != nil {
			t.Errorf("error generating random string: %v", err)
		}
		if len(randomString) != length*2 { // Hex encoding doubles the length
			t.Errorf("expected string length of %d, but got %d", length*2, len(randomString))
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		randReadLock.Lock()
		originalReader := str.RandReader
		str.RandReader = &errorReader{}
		defer func() {
			str.RandReader = originalReader
			randReadLock.Unlock()
		}()

		_, err := str.GenerateRandomBase64String()
		if err == nil {
			t.Error("expected an error for GenerateRandomBase64String, but got none")
		}

		_, err = str.GenerateRandomHexString()
		if err == nil {
			t.Error("expected an error for GenerateRandomHexString, but got none")
		}
	})
}
