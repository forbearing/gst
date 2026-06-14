package database_test

import (
	"strings"
	"testing"
	"time"

	"github.com/forbearing/gst/cache"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/stretchr/testify/require"
)

func TestDatabaseBuildSQL(t *testing.T) {
	t.Run("NilBuilder", func(t *testing.T) {
		stmts, err := database.Database[*TestUser](nil).BuildSQL(nil)

		require.ErrorIs(t, err, database.ErrNilSQLBuilder)
		require.Nil(t, stmts)
	})

	t.Run("List", func(t *testing.T) {
		users := make([]*TestUser, 0)

		stmts, err := database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name}).
			WithOrder("created_at DESC").
			BuildSQL(func(db types.Database[*TestUser]) error {
				return db.List(&users)
			})

		require.NoError(t, err)
		require.Len(t, stmts, 1)
		requireSQLContains(t, stmts[0], "SELECT", "FROM", "test_users", "WHERE", "ORDER BY")
		require.Contains(t, stmts[0].Vars, u1.Name)
		require.Len(t, users, 0, "BuildSQL should not execute the query or fill the destination")
	})

	t.Run("CreateDoesNotExecute", func(t *testing.T) {
		defer cleanupTestData()

		user := &TestUser{Name: "build-sql-create", Email: "build-sql-create@example.com"}
		stmts, err := database.Database[*TestUser](nil).BuildSQL(func(db types.Database[*TestUser]) error {
			return db.Create(user)
		})

		require.NoError(t, err)
		require.Len(t, stmts, 1)
		requireSQLContains(t, stmts[0], "INSERT", "INTO", "test_users")
		require.Contains(t, stmts[0].Vars, user.Name)
		require.Empty(t, user.ID, "BuildSQL should not fill model IDs")
		require.Nil(t, user.CreatedAt, "BuildSQL should not fill created_at")
		require.Nil(t, user.UpdatedAt, "BuildSQL should not fill updated_at")
		require.Nil(t, user.Remark, "BuildSQL should not run model hooks")

		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: user.Name}).
			List(&users))
		require.Len(t, users, 0, "BuildSQL should not create database rows")
	})

	t.Run("BatchCreate", func(t *testing.T) {
		users := []*TestUser{
			{Name: "build-sql-batch-1", Email: "build-sql-batch-1@example.com"},
			{Name: "build-sql-batch-2", Email: "build-sql-batch-2@example.com"},
			{Name: "build-sql-batch-3", Email: "build-sql-batch-3@example.com"},
		}

		stmts, err := database.Database[*TestUser](nil).
			WithBatchSize(2).
			BuildSQL(func(db types.Database[*TestUser]) error {
				return db.Create(users...)
			})

		require.NoError(t, err)
		require.Len(t, stmts, 2)
		requireSQLContains(t, stmts[0], "INSERT", "INTO", "test_users")
		requireSQLContains(t, stmts[1], "INSERT", "INTO", "test_users")
		require.Contains(t, stmts[0].Vars, users[0].Name)
		require.Contains(t, stmts[0].Vars, users[1].Name)
		require.Contains(t, stmts[1].Vars, users[2].Name)
	})

	t.Run("TransactionUnsupported", func(t *testing.T) {
		stmts, err := database.Database[*TestUser](nil).BuildSQL(func(db types.Database[*TestUser]) error {
			return db.Transaction(func(txDB types.Database[*TestUser]) error {
				return nil
			})
		})

		require.ErrorIs(t, err, database.ErrBuildSQLTransaction)
		require.Len(t, stmts, 0)
	})

	t.Run("TransactionFuncUnsupported", func(t *testing.T) {
		stmts, err := database.Database[*TestUser](nil).BuildSQL(func(db types.Database[*TestUser]) error {
			return db.TransactionFunc(func(tx any) error {
				return nil
			})
		})

		require.ErrorIs(t, err, database.ErrBuildSQLTransaction)
		require.Len(t, stmts, 0)
	})

	t.Run("GetIgnoresDestinationID", func(t *testing.T) {
		existingID := u1.ID
		requestedID := u2.ID
		dest := &TestUser{Base: model.Base{ID: existingID}}

		stmts, err := database.Database[*TestUser](nil).BuildSQL(func(db types.Database[*TestUser]) error {
			return db.Get(dest, requestedID)
		})

		require.NoError(t, err)
		require.Len(t, stmts, 1)
		requireSQLContains(t, stmts[0], "SELECT", "FROM", "test_users", "WHERE")
		require.Equal(t, []any{requestedID}, stmts[0].Vars, "Get SQL should only use the requested id")
		require.Equal(t, existingID, dest.ID, "BuildSQL should leave destination values unchanged")
	})

	t.Run("WithDryRunAndCache", func(t *testing.T) {
		defer cleanupTestData()

		listCache := cache.Cache[[]*TestUser]()
		modelCache := cache.Cache[*TestUser]()
		listCache.Clear()
		modelCache.Clear()
		defer listCache.Clear()
		defer modelCache.Clear()

		cachedList := []*TestUser{{Name: "build-sql-cached-list"}}
		cachedUser := &TestUser{Name: "build-sql-cached-user"}
		require.NoError(t, listCache.Set("build-sql-list-cache", cachedList, time.Minute))
		require.NoError(t, modelCache.Set(u1.ID, cachedUser, time.Minute))

		user := &TestUser{Name: "build-sql-dry-run-cache", Email: "build-sql-dry-run-cache@example.com"}
		stmts, err := database.Database[*TestUser](nil).
			WithCache().
			WithDryRun().
			BuildSQL(func(db types.Database[*TestUser]) error {
				return db.Create(user)
			})

		require.NoError(t, err)
		require.Len(t, stmts, 1)
		requireSQLContains(t, stmts[0], "INSERT", "INTO", "test_users")
		require.Contains(t, stmts[0].Vars, user.Name)
		require.Empty(t, user.ID, "BuildSQL with dry-run should not fill model IDs")
		require.Nil(t, user.CreatedAt, "BuildSQL with dry-run should not fill created_at")
		require.Nil(t, user.UpdatedAt, "BuildSQL with dry-run should not fill updated_at")
		require.Nil(t, user.Remark, "BuildSQL with dry-run should not run model hooks")

		gotList, err := listCache.Get("build-sql-list-cache")
		require.NoError(t, err, "BuildSQL with dry-run should not clear list cache")
		require.Equal(t, cachedList, gotList)

		gotUser, err := modelCache.Get(u1.ID)
		require.NoError(t, err, "BuildSQL with dry-run should not delete model cache")
		require.Equal(t, cachedUser, gotUser)

		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: user.Name}).
			List(&users))
		require.Len(t, users, 0, "BuildSQL with dry-run should not create database rows")
	})
}

func requireSQLContains(t *testing.T, stmt types.SQLStatement, parts ...string) {
	t.Helper()

	sql := strings.ToUpper(stmt.SQL)
	for _, part := range parts {
		require.Contains(t, sql, strings.ToUpper(part))
	}
}
