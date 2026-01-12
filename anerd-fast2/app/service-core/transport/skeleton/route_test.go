package skeleton_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	proto "gofast/gen/proto/v1"
	"gofast/pkg/auth"
	pkgtest "gofast/pkg/testutil"
	skeletonSvc "gofast/service-core/domain/skeleton"
	"gofast/service-core/storage/query"
	storetest "gofast/service-core/storage/testutil"
	"gofast/service-core/transport/skeleton"
)

// --- Test Environment ---

type testEnv struct {
	db      *sql.DB
	store   *query.Queries
	deps    skeletonSvc.Deps
	server  *skeleton.Server
	cleanup func()
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	testDB := pkgtest.SetupTestDB(t)
	store := query.New(testDB.DB)
	deps := skeletonSvc.Deps{Store: store}

	return &testEnv{
		db:      testDB.DB,
		store:   store,
		deps:    deps,
		server:  skeleton.NewSkeletonServer(deps),
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

func TestServer_CreateSkeleton(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.CreateSkeleton)
		ctx := contextWithUser(user)

		req := connect.NewRequest(&proto.CreateSkeletonRequest{
			//nolint:exhaustruct
			Skeleton: &proto.Skeleton{
				// GF_TP_TEST_CREATE_FIELDS_START
				Name:   "Test Skeleton",
				Age:    "100",
				Death:  "2023-10-31",
				Zombie: true,
				// GF_TP_TEST_CREATE_FIELDS_END
			},
		})

		res, err := env.server.CreateSkeleton(ctx, req)

		require.NoError(t, err)
		assert.NotEmpty(t, res.Msg.GetSkeleton().GetId())
	})
}

func TestServer_GetSkeletonByID(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		ctx := contextWithUser(user)
		skel := createTestSkeleton(t, env, user.ID)

		req := connect.NewRequest(&proto.GetSkeletonByIDRequest{
			Id: skel.ID.String(),
		})

		res, err := env.server.GetSkeletonByID(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, skel.ID.String(), res.Msg.GetSkeleton().GetId())
	})
}

func TestServer_EditSkeleton(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.EditSkeleton)
		ctx := contextWithUser(user)
		skel := createTestSkeleton(t, env, user.ID)

		req := connect.NewRequest(&proto.EditSkeletonRequest{
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
		})

		res, err := env.server.EditSkeleton(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, skel.ID.String(), res.Msg.GetSkeleton().GetId())
	})
}

func TestServer_RemoveSkeleton(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.RemoveSkeleton)
		ctx := contextWithUser(user)
		skel := createTestSkeleton(t, env, user.ID)

		req := connect.NewRequest(&proto.RemoveSkeletonRequest{
			Id: skel.ID.String(),
		})

		_, err := env.server.RemoveSkeleton(ctx, req)
		require.NoError(t, err)

		// Verify skeleton is removed
		//nolint:exhaustruct
		getCtx := contextWithUser(query.User{
			ID:     user.ID,
			Access: auth.UserAccess | auth.GetSkeletons,
			Avatar: user.Avatar,
			Email:  user.Email,
		})
		getReq := connect.NewRequest(&proto.GetSkeletonByIDRequest{
			Id: skel.ID.String(),
		})
		_, err = env.server.GetSkeletonByID(getCtx, getReq)
		assert.Error(t, err)
	})
}

func TestServer_GetAllSkeletonsLogic(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		env := setupTestEnv(t)
		defer env.cleanup()

		user := storetest.CreateTestUser(t, env.store, auth.UserAccess|auth.GetSkeletons)
		ctx := contextWithUser(user)

		createTestSkeleton(t, env, user.ID)
		createTestSkeleton(t, env, user.ID)

		var sent []*proto.GetAllSkeletonsResponse
		err := env.server.GetAllSkeletonsLogic(ctx, func(resp *proto.GetAllSkeletonsResponse) error {
			sent = append(sent, resp)
			return nil
		})

		require.NoError(t, err)
		assert.Len(t, sent, 2)
	})
}
