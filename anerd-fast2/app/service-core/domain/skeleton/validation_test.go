package skeleton_test

import (
	"gofast/pkg"
	"gofast/service-core/domain/skeleton"
	"testing"

	proto "gofast/gen/proto/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// GF_FIXTURES_START
// Helpers to construct proto payloads used across tests.
// The generator will replace these per model.
func makeCreateSkeletonProto(name string, age string, death string, zombie bool) *proto.Skeleton {
	return &proto.Skeleton{
		Id:      "",
		Created: "",
		Updated: "",
		Name:    name,
		Age:     age,
		Death:   death,
		Zombie:  zombie,
	}
}

func makeEditSkeletonProto(id string, name string, age string, death string, zombie bool) *proto.Skeleton {
	return &proto.Skeleton{
		Id:      id,
		Created: "",
		Updated: "",
		Name:    name,
		Age:     age,
		Death:   death,
		Zombie:  zombie,
	}
}

// GF_FIXTURES_END

func TestValidateAndBuildInsertParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		skeleton       *proto.Skeleton
		expectError    bool
		expectedErrors []pkg.ValidationError
	}{
		{
			name:           "valid skeleton",
			skeleton:       makeCreateSkeletonProto("Skelly", "10", "2025-01-01", true),
			expectError:    false,
			expectedErrors: nil,
		},
		{
			name:        "empty name",
			skeleton:    makeCreateSkeletonProto("", "10", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "name", Tag: "required", Message: "Name is required"},
			},
		},
		{
			name:        "name too short",
			skeleton:    makeCreateSkeletonProto("Sk", "10", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "name", Tag: "minlength", Message: "Name must be at least 3 characters long"},
			},
		},
		{
			name:        "age is not a number",
			skeleton:    makeCreateSkeletonProto("Skelly", "ten", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "age", Tag: "number", Message: "Age must be a number"},
			},
		},
		{
			name:        "age less than 1",
			skeleton:    makeCreateSkeletonProto("Skelly", "0", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "age", Tag: "gte", Message: "Age must be greater than or equal to 1"},
			},
		},
		{
			name:        "invalid death date",
			skeleton:    makeCreateSkeletonProto("Skelly", "10", "invalid-date", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "death", Tag: "required", Message: "Death date is required and must be in YYYY-MM-DD or RFC3339 format"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			userID := uuid.New()
			params, validationErrors := skeleton.ValidateAndBuildInsertParams(userID, tc.skeleton)
			if tc.expectError {
				assert.NotNil(t, validationErrors)
				assert.Equal(t, tc.expectedErrors, validationErrors)
				assert.Nil(t, params)
			} else {
				assert.Nil(t, validationErrors)
				assert.NotNil(t, params)
				assert.Equal(t, userID, params.UserID)
			}
		})
	}
}

func TestValidateAndBuildUpdateParams(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		skeleton       *proto.Skeleton
		expectError    bool
		expectedErrors []pkg.ValidationError
	}{
		{
			name:           "valid skeleton",
			skeleton:       makeEditSkeletonProto(uuid.New().String(), "Skelly", "10", "2025-01-01", true),
			expectError:    false,
			expectedErrors: nil,
		},
		{
			name:        "empty name",
			skeleton:    makeEditSkeletonProto(uuid.New().String(), "", "10", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "name", Tag: "required", Message: "Name is required"},
			},
		},
		{
			name:        "invalid uuid",
			skeleton:    makeEditSkeletonProto("invalid-uuid", "Skelly", "10", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "id", Tag: "uuid", Message: "ID must be a valid UUID"},
				{Field: "id", Tag: "required", Message: "ID is required"},
			},
		},
		{
			name:        "nil uuid",
			skeleton:    makeEditSkeletonProto(uuid.Nil.String(), "Skelly", "10", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "id", Tag: "required", Message: "ID is required"},
			},
		},
		{
			name:        "name too short",
			skeleton:    makeEditSkeletonProto(uuid.New().String(), "Sk", "10", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "name", Tag: "minlength", Message: "Name must be at least 3 characters long"},
			},
		},
		{
			name:        "age is not a number",
			skeleton:    makeEditSkeletonProto(uuid.New().String(), "Skelly", "ten", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "age", Tag: "number", Message: "Age must be a number"},
			},
		},
		{
			name:        "age less than 1",
			skeleton:    makeEditSkeletonProto(uuid.New().String(), "Skelly", "0", "2025-01-01", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "age", Tag: "gte", Message: "Age must be greater than or equal to 1"},
			},
		},
		{
			name:        "invalid death date",
			skeleton:    makeEditSkeletonProto(uuid.New().String(), "Skelly", "10", "invalid-date", false),
			expectError: true,
			expectedErrors: []pkg.ValidationError{
				{Field: "death", Tag: "required", Message: "Death date is required and must be in YYYY-MM-DD or RFC3339 format"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			userID := uuid.New()
			params, validationErrors := skeleton.ValidateAndBuildUpdateParams(userID, tc.skeleton)
			if tc.expectError {
				assert.NotNil(t, validationErrors)
				assert.Equal(t, tc.expectedErrors, validationErrors)
				assert.Nil(t, params)
			} else {
				assert.Nil(t, validationErrors)
				assert.NotNil(t, params)
				assert.Equal(t, userID, params.UserID)
			}
		})
	}
}
