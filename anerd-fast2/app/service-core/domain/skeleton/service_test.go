package skeleton_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	proto "gofast/gen/proto/v1"
	"gofast/pkg/auth"
	pkgtest "gofast/pkg/testutil"
	"gofast/service-core/domain/skeleton"
	"gofast/service-core/storage/query"
	storetest "gofast/service-core/storage/testutil"
)

// --- Test Environment ---

type testEnv struct {
	db      *sql.DB
	store   *query.Queries
	deps    skeleton.Deps
	cleanup func()
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	testDB := pkgtest.SetupTestDB(t)
	store := query.New(testDB.DB)

	return &testEnv{
		db:      testDB.DB,
		store:   store,
		deps:    skeleton.Deps{Store: store},
		cleanup: testDB.Cleanup,
	}
}

// --- Test Helpers ---

func createTestSkeleton(t *testing.T, env *testEnv, userID uuid.UUID) query.Skeleton {
	t.Helper()
	ctx := context.Background()

	skel, err := env.store.InsertSkeleton(ctx, query.InsertSkeletonParams{
		UserID: userID,
		// GF_TP_TEST_ENTITY_FIELDS_START
		Name:   "Skeleton " + uuid.New().String()[:8],
		Age:    "100",
		Death:  time.Now(),
		Zombie: true,
		// GF_TP_TEST_ENTITY_FIELDS_END
	})
	require.NoError(t, err)
	return skel
}

func contextWithUser(user query.User) context.Context {
	return auth.NewContextWithUser(context.Background(), &auth.AccessTokenClaims{
		ID:     user.ID,
		Access: user.Access,
		Avatar: user.Avatar,
		Email:  user.Email,
	})
}

// --- Tests ---

func TestService_CreateSkeleton(t *testing.T) {
	t.Parallel()
	t.Run("Failure - Unauthorized (no claims)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background() // No claims in context

		req := &proto.CreateSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				// GF_TP_TEST_CREATE_FIELDS_START
				Name:   "Test Skeleton",
				Age:    "100",
				Death:  "2023-10-31",
				Zombie: true,
				// GF_TP_TEST_CREATE_FIELDS_END
			},
		}

		_, err := skeleton.CreateSkeleton(ctx, &env.deps, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no claims found in context")
	})

	t.Run("Failure - Validation Error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.CreateSkeleton)
		ctx := contextWithUser(user)

		req := &proto.CreateSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				// GF_TP_TEST_INVALID_FIELDS_START
				Name:  "",
				Age:   "invalid",
				Death: "bad-date",
				// GF_TP_TEST_INVALID_FIELDS_END
			},
		}

		_, err := skeleton.CreateSkeleton(ctx, &env.deps, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation errors")
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.CreateSkeleton)
		ctx := contextWithUser(user)

		req := &proto.CreateSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				// GF_TP_TEST_CREATE_FIELDS_START
				Name:   "Test Skeleton",
				Age:    "100",
				Death:  "2023-10-31",
				Zombie: true,
				// GF_TP_TEST_CREATE_FIELDS_END
			},
		}

		created, err := skeleton.CreateSkeleton(ctx, &env.deps, req)

		require.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, user.ID, created.UserID)
	})

	t.Run("Failure - Forbidden", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.BasicPlan)
		ctx := contextWithUser(user)

		req := &proto.CreateSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				// GF_TP_TEST_CREATE_FIELDS_START
				Name:   "Test Skeleton",
				Age:    "100",
				Death:  "2023-10-31",
				Zombie: true,
				// GF_TP_TEST_CREATE_FIELDS_END
			},
		}

		_, err := skeleton.CreateSkeleton(ctx, &env.deps, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})
}

func TestService_GetSkeletonByID(t *testing.T) {
	t.Parallel()
	t.Run("Failure - Unauthorized (no claims)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background() // No claims in context

		_, err := skeleton.GetSkeletonByID(ctx, &env.deps, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no claims found in context")
	})

	t.Run("Failure - Forbidden", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.BasicPlan) // No GetSkeletons permission
		ctx := contextWithUser(user)

		_, err := skeleton.GetSkeletonByID(ctx, &env.deps, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		ctx := contextWithUser(user)
		skel := createTestSkeleton(t, env, user.ID)

		fetched, err := skeleton.GetSkeletonByID(ctx, &env.deps, skel.ID)

		require.NoError(t, err)
		assert.Equal(t, skel.ID, fetched.ID)
	})

	t.Run("Failure - Not Found (Wrong User)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user1 := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		user2 := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		skel := createTestSkeleton(t, env, user1.ID)

		ctx := contextWithUser(user2)
		_, err := skeleton.GetSkeletonByID(ctx, &env.deps, skel.ID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sql: no rows")
	})
}

func TestService_GetAllSkeletons(t *testing.T) {
	t.Parallel()
	t.Run("Failure - Unauthorized (no claims)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background() // No claims in context

		err := skeleton.GetAllSkeletons(ctx, &env.deps, func(_ context.Context, _ *query.Skeleton) error {
			return nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no claims found in context")
	})

	t.Run("Failure - Forbidden", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.BasicPlan) // No GetSkeletons permission
		ctx := contextWithUser(user)

		err := skeleton.GetAllSkeletons(ctx, &env.deps, func(_ context.Context, _ *query.Skeleton) error {
			return nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("Failure - Processor Error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		ctx := contextWithUser(user)
		createTestSkeleton(t, env, user.ID)

		err := skeleton.GetAllSkeletons(ctx, &env.deps, func(_ context.Context, _ *query.Skeleton) error {
			return errors.New("processor failed")
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "processor failed")
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		ctx := contextWithUser(user)

		createTestSkeleton(t, env, user.ID)
		createTestSkeleton(t, env, user.ID)
		createTestSkeleton(t, env, user.ID)

		count := 0
		err := skeleton.GetAllSkeletons(ctx, &env.deps, func(_ context.Context, _ *query.Skeleton) error {
			count++
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

func TestService_EditSkeleton(t *testing.T) {
	t.Parallel()
	t.Run("Failure - Unauthorized (no claims)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background() // No claims in context

		req := &proto.EditSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				Id: uuid.New().String(),
				// GF_TP_TEST_EDIT_FIELDS_START
				Name:   "Updated Name",
				Age:    "200",
				Death:  "2024-01-01",
				Zombie: false,
				// GF_TP_TEST_EDIT_FIELDS_END
			},
		}

		_, err := skeleton.EditSkeleton(ctx, &env.deps, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no claims found in context")
	})

	t.Run("Failure - Forbidden", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.BasicPlan) // No EditSkeleton permission
		ctx := contextWithUser(user)

		req := &proto.EditSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				Id: uuid.New().String(),
				// GF_TP_TEST_EDIT_FIELDS_START
				Name:   "Updated Name",
				Age:    "200",
				Death:  "2024-01-01",
				Zombie: false,
				// GF_TP_TEST_EDIT_FIELDS_END
			},
		}

		_, err := skeleton.EditSkeleton(ctx, &env.deps, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("Failure - Validation Error", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.EditSkeleton)
		ctx := contextWithUser(user)

		req := &proto.EditSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				Id: "invalid-uuid",
				// GF_TP_TEST_INVALID_FIELDS_START
				Name:  "",
				Age:   "invalid",
				Death: "bad-date",
				// GF_TP_TEST_INVALID_FIELDS_END
			},
		}

		_, err := skeleton.EditSkeleton(ctx, &env.deps, req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation errors")
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.EditSkeleton)
		ctx := contextWithUser(user)
		skel := createTestSkeleton(t, env, user.ID)

		req := &proto.EditSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				Id: skel.ID.String(),
				// GF_TP_TEST_EDIT_FIELDS_START
				Name:   "Updated Name",
				Age:    "200",
				Death:  "2024-01-01",
				Zombie: false,
				// GF_TP_TEST_EDIT_FIELDS_END
			},
		}

		updated, err := skeleton.EditSkeleton(ctx, &env.deps, req)

		require.NoError(t, err)
		assert.NotNil(t, updated)
	})
}

func TestService_RemoveSkeleton(t *testing.T) {
	t.Parallel()
	t.Run("Failure - Unauthorized (no claims)", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		ctx := context.Background() // No claims in context

		err := skeleton.RemoveSkeleton(ctx, &env.deps, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no claims found in context")
	})

	t.Run("Failure - Forbidden", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.BasicPlan) // No RemoveSkeleton permission
		ctx := contextWithUser(user)

		err := skeleton.RemoveSkeleton(ctx, &env.deps, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.RemoveSkeleton)
		ctx := contextWithUser(user)
		skel := createTestSkeleton(t, env, user.ID)

		err := skeleton.RemoveSkeleton(ctx, &env.deps, skel.ID)
		require.NoError(t, err)

		_, err = env.store.SelectSkeletonByID(ctx, query.SelectSkeletonByIDParams{
			ID:     skel.ID,
			UserID: user.ID,
		})
		require.Error(t, err)
	})
}
