package database_test

import (
	"testing"

	"github.com/forbearing/gst/database"
	"github.com/stretchr/testify/require"
)

func TestDatabaseCleanup(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Verify initial count - should have 3 records
	count := new(int64)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(3), *count)

	// Soft delete some records (u1 and u2)
	require.NoError(t, database.Database[*TestUser](nil).Delete(u1, u2))

	// Verify soft-deleted records are not visible in normal queries
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(1), *count, "only u3 should be visible after soft delete")

	// Verify u3 is still accessible
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u3.ID))
	require.NotNil(t, u)
	require.Equal(t, u3.ID, u.ID)
	require.Equal(t, u3.Name, u.Name)

	// Test Cleanup - should permanently delete soft-deleted records (u1 and u2)
	require.NoError(t, database.Database[*TestUser](nil).Cleanup())

	// Verify soft-deleted records are permanently removed
	// After Cleanup, u1 and u2 should be permanently deleted
	// u3 should still exist
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(1), *count, "u3 should still exist after Cleanup")

	// Verify u3 is still accessible
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u3.ID))
	require.NotNil(t, u)
	require.Equal(t, u3.ID, u.ID)
	require.Equal(t, u3.Name, u.Name)
	require.Equal(t, u3.Age, u.Age)
	require.Equal(t, u3.Email, u.Email)

	// Test Cleanup with no soft-deleted records - should not error
	require.NoError(t, database.Database[*TestUser](nil).Cleanup())

	// Verify u3 still exists after second Cleanup
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(1), *count, "u3 should still exist after second Cleanup")

	// Test Cleanup with different model types
	require.NoError(t, database.Database[*TestProduct](nil).Cleanup())
	require.NoError(t, database.Database[*TestCategory](nil).Cleanup())
}

func TestDatabaseHealth(t *testing.T) {
	// Test basic health check - should pass when database is healthy
	require.NoError(t, database.Database[*TestUser](nil).Health())

	// Test health check multiple times - should be idempotent
	require.NoError(t, database.Database[*TestUser](nil).Health())
	require.NoError(t, database.Database[*TestUser](nil).Health())

	// Test health check after database operations - should still pass
	defer cleanupTestData()
	setupTestData(t)
	require.NoError(t, database.Database[*TestUser](nil).Health())

	// Test health check with different model types - should work for all models
	require.NoError(t, database.Database[*TestProduct](nil).Health())
	require.NoError(t, database.Database[*TestCategory](nil).Health())
}
