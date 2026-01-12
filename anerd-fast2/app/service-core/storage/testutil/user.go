package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"gofast/pkg/auth"
	"gofast/service-core/storage/query"
)

// CreateTestUser creates a test user with an active subscription.
// If access is not provided, defaults to auth.UserAccess.
func CreateTestUser(t *testing.T, store *query.Queries, access ...int64) query.User {
	t.Helper()
	ctx := context.Background()

	userAccess := auth.UserAccess
	if len(access) > 0 {
		userAccess = access[0]
	}

	user, err := store.InsertUser(ctx, query.InsertUserParams{
		Email:  fmt.Sprintf("user-%s@test.example.com", uuid.New().String()[:8]),
		Access: userAccess,
		Sub:    "google:" + uuid.New().String(),
		Avatar: "https://example.com/avatar.png",
		ApiKey: uuid.New().String(),
	})
	require.NoError(t, err)

	return user
}
