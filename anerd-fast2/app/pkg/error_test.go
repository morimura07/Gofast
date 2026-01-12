package pkg_test

import (
	"errors"
	"gofast/pkg"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError(t *testing.T) {
	t.Parallel()
	err := pkg.ValidationError{
		Field:   "email",
		Tag:     "required",
		Message: "Email is required",
	}
	assert.Equal(t, "Email is required", err.Error())
}

func TestValidationErrors(t *testing.T) {
	t.Parallel()
	errs := pkg.ValidationErrors{
		pkg.ValidationError{Field: "email", Tag: "required", Message: "Email is required"},
		pkg.ValidationError{Field: "password", Tag: "min", Message: "Password must be at least 8 characters"},
	}
	expected := `[{"field": "email", "tag": "required", "message": "Email is required"}, {"field": "password", "tag": "min", "message": "Password must be at least 8 characters"}]`
	assert.Equal(t, expected, errs.Error())
}

func TestCustomErrors(t *testing.T) {
	t.Parallel()
	baseErr := errors.New("database connection failed")

	testCases := []struct {
		name           string
		err            error
		expectedMsg    string
		expectedUnwrap error
	}{
		{
			name:           "InternalError",
			err:            pkg.InternalError{Err: baseErr},
			expectedMsg:    "internal error: database connection failed",
			expectedUnwrap: baseErr,
		},
		{
			name:           "BadRequestError",
			err:            pkg.BadRequestError{Err: baseErr},
			expectedMsg:    "bad request: database connection failed",
			expectedUnwrap: baseErr,
		},
		{
			name:           "NotFoundError",
			err:            pkg.NotFoundError{Err: baseErr},
			expectedMsg:    "not found: database connection failed",
			expectedUnwrap: baseErr,
		},
		{
			name:           "UnauthorizedError",
			err:            pkg.UnauthorizedError{Err: baseErr},
			expectedMsg:    "unauthorized: database connection failed",
			expectedUnwrap: baseErr,
		},
		{
			name:           "ForbiddenError",
			err:            pkg.ForbiddenError{Err: baseErr},
			expectedMsg:    "forbidden: database connection failed",
			expectedUnwrap: baseErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expectedMsg, tc.err.Error())
			unwrappedErr := errors.Unwrap(tc.err)
			assert.Equal(t, tc.expectedUnwrap, unwrappedErr)
		})
	}
}

func TestCustomErrorsWithoutUnderlyingError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "InternalError",
			err:         pkg.InternalError{},
			expectedMsg: "internal error",
		},
		{
			name:        "BadRequestError",
			err:         pkg.BadRequestError{},
			expectedMsg: "bad request",
		},
		{
			name:        "NotFoundError",
			err:         pkg.NotFoundError{},
			expectedMsg: "not found",
		},
		{
			name:        "UnauthorizedError",
			err:         pkg.UnauthorizedError{},
			expectedMsg: "unauthorized",
		},
		{
			name:        "ForbiddenError",
			err:         pkg.ForbiddenError{},
			expectedMsg: "forbidden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expectedMsg, tc.err.Error())
			unwrappedErr := errors.Unwrap(tc.err)
			assert.Nil(t, unwrappedErr)
		})
	}
}
