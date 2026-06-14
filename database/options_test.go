package database_test

import (
	"os"
	"testing"
	"time"

	"github.com/forbearing/gst/cache"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/database/sqlite"
	"github.com/forbearing/gst/model"
	"github.com/stretchr/testify/require"
)

func TestDatabaseWithDB(t *testing.T) {
	path2 := "/tmp/test2.db"
	path3 := "/tmp/test3.db"
	defer func() {
		_ = os.Remove(path2)
		_ = os.Remove(path3)
		cleanupTestData()
	}()

	users := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users))

	db2, err := sqlite.New(config.Sqlite{
		Enable:   true,
		Path:     path2,
		IsMemory: false,
	})
	require.NoError(t, err)
	db3, err := sqlite.New(config.Sqlite{
		Enable:   true,
		Path:     path3,
		IsMemory: false,
	})
	require.NoError(t, err)

	require.NoError(t, db2.AutoMigrate(TestUser{}))
	require.NoError(t, db3.AutoMigrate(TestUser{}))

	// List from the custom sqlite. the new sqlite db is always empty.
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
	require.Equal(t, 0, len(users))

	// Create resources in db2
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Create(ul...))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.Equal(t, 3, len(users))
	// Verify data integrity
	var foundU1, foundU2, foundU3 bool
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			foundU1 = true
			require.Equal(t, u1.Name, u.Name)
			require.Equal(t, u1.Age, u.Age)
			require.Equal(t, u1.Email, u.Email)
		case u2.ID:
			foundU2 = true
			require.Equal(t, u2.Name, u.Name)
		case u3.ID:
			foundU3 = true
			require.Equal(t, u3.Name, u.Name)
		}
	}
	require.True(t, foundU1 && foundU2 && foundU3, "all users should be found in db2")
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
	require.Equal(t, 0, len(users), "db3 should be empty")

	// Get operation with custom DB
	user := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Get(user, u1.ID))
	require.NotNil(t, user)
	require.Equal(t, u1.ID, user.ID)
	require.Equal(t, u1.Name, user.Name)

	// Update operation with custom DB
	user.Name = "user1_updated"
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Update(user))
	updatedUser := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Get(updatedUser, u1.ID))
	require.Equal(t, "user1_updated", updatedUser.Name)
	user.Name = "user1" // restore

	// Create resources in db3
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).Create(ul...))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.Equal(t, 3, len(users), "db2 should still have 3 users")
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
	require.Equal(t, 3, len(users), "db3 should have 3 users")

	// Delete resources in default db
	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.Equal(t, 3, len(users), "db2 should still have users")
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
	require.Equal(t, 3, len(users), "db3 should still have users")

	// Delete resources in db2
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Delete(ul...))
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
	require.Equal(t, 3, len(users), "db3 should still have users")

	// Delete resources in db3
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).Delete(ul...))
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.Equal(t, 0, len(users))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
	require.Equal(t, 0, len(users))

	// Chainable with other methods
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithQuery(&TestUser{Name: u1.Name}).Create(u1))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
	require.GreaterOrEqual(t, len(users), 1, "should find created user")
}

func TestDatabaseWithTable(t *testing.T) {
	path2 := "/tmp/test2.db"
	path3 := "/tmp/test3.db"
	defer func() {
		_ = os.Remove(path2)
		_ = os.Remove(path3)
	}()

	db2, err := sqlite.New(config.Sqlite{
		Enable:   true,
		Path:     path2,
		IsMemory: false,
	})
	require.NoError(t, err)
	db3, err := sqlite.New(config.Sqlite{
		Enable:   true,
		Path:     path3,
		IsMemory: false,
	})
	require.NoError(t, err)

	// WithTable will not auto migrate the database.
	require.Error(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Create(u1))
	require.Error(t, database.Database[*TestUser](nil).WithDB(db3).WithTable("test_users").Create(ul...))

	// Manually migrate the database.
	require.NoError(t, db2.AutoMigrate(TestUser{}))
	require.NoError(t, db3.AutoMigrate(TestUser{}))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Create(ul...))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).WithTable("test_users").Create(ul...))

	users := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").List(&users))
	require.Equal(t, 3, len(users))
	// Verify data integrity
	var foundU1, foundU2, foundU3 bool
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			foundU1 = true
			require.Equal(t, u1.Name, u.Name)
			require.Equal(t, u1.Age, u.Age)
			require.Equal(t, u1.Email, u.Email)
		case u2.ID:
			foundU2 = true
			require.Equal(t, u2.Name, u.Name)
		case u3.ID:
			foundU3 = true
			require.Equal(t, u3.Name, u.Name)
		}
	}
	require.True(t, foundU1 && foundU2 && foundU3, "all users should be found")

	require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).WithTable("test_users").List(&users))
	require.Equal(t, 3, len(users))

	// Get operation with custom table
	user := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Get(user, u1.ID))
	require.NotNil(t, user)
	require.Equal(t, u1.ID, user.ID)
	require.Equal(t, u1.Name, user.Name)

	// Update operation with custom table
	user.Name = "user1_updated"
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Update(user))
	updatedUser := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Get(updatedUser, u1.ID))
	require.Equal(t, "user1_updated", updatedUser.Name)

	// Delete operation with custom table
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Delete(u1))
	require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").List(&users))
	require.Equal(t, 2, len(users), "should have 2 users after deleting u1")

	// Chainable with other methods
	require.NoError(t, database.Database[*TestUser](nil).
		WithDB(db2).
		WithTable("test_users").
		WithQuery(&TestUser{Name: u2.Name}).
		Get(user, u2.ID))
	require.NotNil(t, user)
	require.Equal(t, u2.ID, user.ID)
}

func TestDatabaseWithBatchSize(t *testing.T) {
	defer cleanupTestData()

	t.Run("Create", func(t *testing.T) {
		t.Run("batch_size_1", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1).Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			// Verify data integrity
			var foundU1, foundU2, foundU3 bool
			for _, u := range users {
				switch u.ID {
				case u1.ID:
					foundU1 = true
					require.Equal(t, u1.Name, u.Name)
					require.Equal(t, u1.Age, u.Age)
					require.Equal(t, u1.Email, u.Email)
				case u2.ID:
					foundU2 = true
					require.Equal(t, u2.Name, u.Name)
				case u3.ID:
					foundU3 = true
					require.Equal(t, u3.Name, u.Name)
				}
			}
			require.True(t, foundU1 && foundU2 && foundU3, "all users should be found after batch create")
		})

		t.Run("batch_size_2", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(2).Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
		})

		t.Run("batch_size_1000", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1000).Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
		})

		t.Run("batch_size_0", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(0).Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users), "should use default batch size when set to 0")
		})

		t.Run("batch_size_negative", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(-1).Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users), "should use default batch size when set to negative")
		})
	})

	t.Run("Update", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		t.Run("batch_size_1", func(t *testing.T) {
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			users[0].Age = 25
			users[1].Age = 26
			users[2].Age = 27
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1).Update(users...))
			users = make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			require.Equal(t, 25, users[0].Age)
			require.Equal(t, 26, users[1].Age)
			require.Equal(t, 27, users[2].Age)
		})

		t.Run("batch_size_2", func(t *testing.T) {
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			users[0].Age = 30
			users[1].Age = 31
			users[2].Age = 32
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(2).Update(users...))
			users = make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			require.Equal(t, 30, users[0].Age)
			require.Equal(t, 31, users[1].Age)
			require.Equal(t, 32, users[2].Age)
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("batch_size_1", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1).Delete(users[0]))
			users = make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 2, len(users))
		})

		t.Run("batch_size_2", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 3, len(users))
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(2).Delete(users...))
			users = make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 0, len(users))
		})

		t.Run("batch_size_large", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(10000).Delete(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Equal(t, 0, len(users))
		})
	})

	t.Run("Combined", func(t *testing.T) {
		defer cleanupTestData()
		require.NoError(t, database.Database[*TestUser](nil).
			WithBatchSize(1000).
			WithQuery(&TestUser{Name: u1.Name}).
			Create(u1))
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.GreaterOrEqual(t, len(users), 1, "should find created user")
	})
}

func TestDatabaseWithDebug(t *testing.T) {
	defer cleanupTestData()

	t.Run("Create", func(t *testing.T) {
		defer cleanupTestData()
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().Create(ul...))
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 3, len(users))
		// Verify data integrity
		var foundU1, foundU2, foundU3 bool
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1 = true
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Age, u.Age)
				require.Equal(t, u1.Email, u.Email)
			case u2.ID:
				foundU2 = true
				require.Equal(t, u2.Name, u.Name)
			case u3.ID:
				foundU3 = true
				require.Equal(t, u3.Name, u.Name)
			}
		}
		require.True(t, foundU1 && foundU2 && foundU3, "all users should be found after debug create")
	})

	t.Run("List", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().List(&users))
		require.Equal(t, 3, len(users))
	})

	t.Run("Get", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		user := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().Get(user, u1.ID))
		require.NotNil(t, user)
		require.Equal(t, u1.ID, user.ID)
		require.Equal(t, u1.Name, user.Name)
		require.Equal(t, u1.Age, user.Age)
		require.Equal(t, u1.Email, user.Email)
	})

	t.Run("Update", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		user := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Get(user, u1.ID))
		user.Age = 25
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().Update(user))
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 3, len(users))
		for _, u := range users {
			if u.ID == u1.ID {
				require.Equal(t, 25, u.Age, "user age should be updated")
			}
		}
	})

	t.Run("Delete", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().Delete(ul...))
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 0, len(users))
	})

	t.Run("Count", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().Count(count))
		require.GreaterOrEqual(t, *count, int64(1), "count should be at least 1")
	})

	t.Run("First", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		firstUser := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).WithDebug().First(firstUser))
		require.NotNil(t, firstUser.ID, "first user should have an ID")
	})

	t.Run("Combined", func(t *testing.T) {
		defer cleanupTestData()
		require.NoError(t, database.Database[*TestUser](nil).
			WithDebug().
			WithQuery(&TestUser{Name: u1.Name}).
			Create(u1))
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.GreaterOrEqual(t, len(users), 1, "should find created user")
	})
}

func TestDatabaseWithDryRun(t *testing.T) {
	defer cleanupTestData()

	t.Run("Create", func(t *testing.T) {
		defer cleanupTestData()

		// WithDryRun should only build the INSERT statement without executing hooks or database I/O.
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Create(ul...))
		require.Nil(t, u1.Remark, "Create should not run model hooks in dry-run mode")
		require.Nil(t, u2.Remark, "Create should not run model hooks in dry-run mode")
		require.Nil(t, u3.Remark, "Create should not run model hooks in dry-run mode")
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Len(t, users, 0, "records should not be created in dry-run mode")

		dryRunUser := &TestUser{Name: "dry-run-create", Email: "dry-run-create@example.com"}
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Create(dryRunUser))
		require.Empty(t, dryRunUser.ID, "Create should not set ID in dry-run mode")
		require.Nil(t, dryRunUser.CreatedAt, "Create should not set created_at in dry-run mode")
		require.Nil(t, dryRunUser.UpdatedAt, "Create should not set updated_at in dry-run mode")
		require.Nil(t, dryRunUser.Remark, "Create should not run model hooks in dry-run mode")
	})

	t.Run("Delete", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should not actually delete records
		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count, "should have 3 records initially")

		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Delete(u1))
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count, "records should not be deleted in dry-run mode")

		softDeleteUser := &dryRunSoftDeleteUser{Name: "dry-run-soft-delete", Base: model.Base{ID: "dry-run-soft-delete"}}
		require.NoError(t, database.Database[*dryRunSoftDeleteUser](nil).WithDryRun().Delete(softDeleteUser))
		require.False(t, softDeleteUser.DeletedAt.Valid, "Delete should not set deleted_at in dry-run mode")
	})

	t.Run("Update", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the UPDATE statement without executing hooks or database I/O.
		originalName := u1.Name
		u1.Name = "updated_name"
		u1.Remark = nil
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Update(u1))
		require.Nil(t, u1.Remark, "Update should not run model hooks in dry-run mode")

		// Verify record is not updated
		uu := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Get(uu, u1.ID))
		require.Equal(t, originalName, uu.Name, "name should not be updated in dry-run mode")

		dryRunUser := &TestUser{Name: "dry-run-update", Email: "dry-run-update@example.com"}
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Update(dryRunUser))
		require.Empty(t, dryRunUser.ID, "Update should not set ID in dry-run mode")
		require.Nil(t, dryRunUser.CreatedAt, "Update should not set created_at in dry-run mode")
		require.Nil(t, dryRunUser.UpdatedAt, "Update should not set updated_at in dry-run mode")
		require.Nil(t, dryRunUser.Remark, "Update should not run model hooks in dry-run mode")
	})

	t.Run("Cache", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		assertDryRunKeepsCache := func(t *testing.T, fn func() error) {
			t.Helper()

			listCache := cache.Cache[[]*TestUser]()
			modelCache := cache.Cache[*TestUser]()
			listCache.Clear()
			modelCache.Clear()
			defer listCache.Clear()
			defer modelCache.Clear()

			cachedList := []*TestUser{{Name: "cached-list"}}
			cachedUser := &TestUser{Name: "cached-user"}
			require.NoError(t, listCache.Set("dry-run-list-cache", cachedList, time.Minute))
			require.NoError(t, modelCache.Set(u1.ID, cachedUser, time.Minute))

			require.NoError(t, fn())

			gotList, err := listCache.Get("dry-run-list-cache")
			require.NoError(t, err, "dry-run should not clear list cache")
			require.Equal(t, cachedList, gotList, "dry-run should leave list cache unchanged")

			gotUser, err := modelCache.Get(u1.ID)
			require.NoError(t, err, "dry-run should not delete model cache")
			require.Equal(t, cachedUser, gotUser, "dry-run should leave model cache unchanged")
		}

		t.Run("Create", func(t *testing.T) {
			assertDryRunKeepsCache(t, func() error {
				return database.Database[*TestUser](nil).WithCache().WithDryRun().Create(ul...)
			})
		})

		t.Run("Delete", func(t *testing.T) {
			assertDryRunKeepsCache(t, func() error {
				return database.Database[*TestUser](nil).WithCache().WithDryRun().Delete(u1)
			})
		})

		t.Run("Update", func(t *testing.T) {
			assertDryRunKeepsCache(t, func() error {
				return database.Database[*TestUser](nil).WithCache().WithDryRun().Update(u1)
			})
		})

		t.Run("UpdateByID", func(t *testing.T) {
			assertDryRunKeepsCache(t, func() error {
				return database.Database[*TestUser](nil).WithCache().WithDryRun().UpdateByID(u1.ID, "name", "updated_name")
			})
		})
	})

	t.Run("UpdateByID", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should not actually update records
		originalName := u1.Name
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().UpdateByID(u1.ID, "name", "updated_name"))

		// Verify record is not updated
		uu := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Get(uu, u1.ID))
		require.Equal(t, originalName, uu.Name, "name should not be updated in dry-run mode")
	})

	t.Run("List", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the SELECT statement without executing it.
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().List(&users))
		require.Len(t, users, 0, "List should not return results in dry-run mode")
	})

	t.Run("Get", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the SELECT statement without executing it.
		uu := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Get(uu, u1.ID))
		require.NotNil(t, uu)
		require.Empty(t, uu.ID, "Get should not return results in dry-run mode")
	})

	t.Run("Count", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the SELECT statement without executing it.
		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Count(count))
		require.Equal(t, int64(0), *count, "Count should not execute in dry-run mode")
	})

	t.Run("First", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the SELECT statement without executing it.
		uu := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().First(uu))
		require.NotNil(t, uu)
		require.Empty(t, uu.ID, "First should not return results in dry-run mode")
	})

	t.Run("Last", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the SELECT statement without executing it.
		uu := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Last(uu))
		require.NotNil(t, uu)
		require.Empty(t, uu.ID, "Last should not return results in dry-run mode")
	})

	t.Run("Take", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// WithDryRun should only build the SELECT statement without executing it.
		uu := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).WithDryRun().Take(uu))
		require.NotNil(t, uu)
		require.Empty(t, uu.ID, "Take should not return results in dry-run mode")
	})
}

type dryRunSoftDeleteUser struct {
	Name string `json:"name"`

	model.Base
}
