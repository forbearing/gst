package database_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/forbearing/gst/bootstrap"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/database/sqlite"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	"github.com/stretchr/testify/require"
)

const (
	remarkUserCreateBefore = "user create before"
	remarkUserUpdateBefore = "user update before"
)

var (
	u1 = &TestUser{Name: "user1", Email: "user1@example.com", Age: 18, Base: model.Base{ID: "u1"}}
	u2 = &TestUser{Name: "user2", Email: "user2@example.com", Age: 19, Base: model.Base{ID: "u2"}}
	u3 = &TestUser{Name: "user3", Email: "user3@example.com", Age: 20, Base: model.Base{ID: "u3"}}

	ul = []*TestUser{u1, u2, u3}
)

// cleanupTestData deletes test data from database and restores original values of test users.
// This function should be called in defer to ensure cleanup after each test.
func cleanupTestData() {
	users := make([]*TestUser, 0)
	_ = database.Database[*TestUser](nil).List(&users)
	_ = database.Database[*TestUser](nil).Delete(users...)
	// Restore original values
	u1 = &TestUser{Name: "user1", Email: "user1@example.com", Age: 18, Base: model.Base{ID: "u1"}}
	u2 = &TestUser{Name: "user2", Email: "user2@example.com", Age: 19, Base: model.Base{ID: "u2"}}
	u3 = &TestUser{Name: "user3", Email: "user3@example.com", Age: 20, Base: model.Base{ID: "u3"}}
	ul = []*TestUser{u1, u2, u3}
}

// setupTestData deletes existing test data and creates all test users (ul).
// This is a common setup pattern used in most test cases.
func setupTestData(t *testing.T) {
	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
	require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
}

// findUsersByID finds users from a slice by their IDs and returns them in order (u1, u2, u3).
// Returns nil for users that are not found.
func findUsersByID(users []*TestUser) (u11, u22, u33 *TestUser) {
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			u11 = u
		case u2.ID:
			u22 = u
		case u3.ID:
			u33 = u
		}
	}
	return
}

type TestUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	IsActive *bool  `json:"is_active"`

	model.Base
}

func (t *TestUser) Purge() bool { return true }
func (t *TestUser) CreateBefore(ctx *types.ModelContext) error {
	t.Remark = util.ValueOf(string(remarkUserCreateBefore))
	return nil
}

func (t *TestUser) UpdateBefore(ctx *types.ModelContext) error {
	t.Remark = util.ValueOf(string(remarkUserUpdateBefore))
	return nil
}

type TestProduct struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CategoryID  string  `json:"category_id"`

	model.Base
}

func (*TestProduct) Purge() bool { return true }

type TestCategory struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`

	model.Base
}

func (*TestCategory) Purge() bool { return true }

func init() {
	os.Setenv(config.LOGGER_DIR, "/tmp/test_database")
	os.Setenv(config.DATABASE_TYPE, string(config.DBSqlite))
	os.Setenv(config.SQLITE_IS_MEMORY, "false")
	os.Setenv(config.SQLITE_PATH, "/tmp/test.db")
	_ = os.Remove("/tmp/test.db")

	os.Setenv(config.DATABASE_TYPE, string(config.DBMySQL))
	os.Setenv(config.MYSQL_DATABASE, "test")
	os.Setenv(config.MYSQL_USERNAME, "test")
	os.Setenv(config.MYSQL_PASSWORD, "test")

	// TODO: test for sqlite, mysql, postgresql

	model.Register[*TestUser]()
	model.Register[*TestProduct]()
	model.Register[*TestCategory]()

	if err := bootstrap.Bootstrap(); err != nil {
		panic(err)
	}
}

// TestDatabase

func TestDatabaseCreate(t *testing.T) {
	defer cleanupTestData()

	// Test basic Create - single record
	require.NoError(t, database.Database[*TestUser](nil).Create(u1))
	count := new(int64)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(1), *count, "should have 1 record after creating single record")

	// Verify single record was created correctly
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.NotNil(t, u)
	require.NotEmpty(t, u.ID, "id should not be empty")
	require.NotEmpty(t, u.CreatedAt, "created_at should not be empty")
	require.NotEmpty(t, u.UpdatedAt, "updated_at should not be empty")
	require.Equal(t, u1.Name, u.Name, "name should match")
	require.Equal(t, u1.Age, u.Age, "age should match")
	require.Equal(t, u1.Email, u.Email, "email should match")

	// Check the create hook result
	require.Equal(t, remarkUserCreateBefore, *u1.Remark, "u1 should have create hook result")

	// Test Create - batch create multiple records
	u1.Remark, u2.Remark, u3.Remark = nil, nil, nil // clear remark to test hook
	require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(3), *count, "should have 3 records after batch create")

	// Check the create hook results for batch create
	require.Equal(t, remarkUserCreateBefore, *u1.Remark, "u1 should have create hook result")
	require.Equal(t, remarkUserCreateBefore, *u2.Remark, "u2 should have create hook result")
	require.Equal(t, remarkUserCreateBefore, *u3.Remark, "u3 should have create hook result")

	// Verify created data in the database
	users := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records")
	var u11, u22, u33 *TestUser
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			u11 = u
		case u2.ID:
			u22 = u
		case u3.ID:
			u33 = u
		}
	}
	require.NotNil(t, u11, "u1 should be found")
	require.NotNil(t, u22, "u2 should be found")
	require.NotNil(t, u33, "u3 should be found")
	require.NotEmpty(t, u11.CreatedAt, "u1 created_at should not be empty")
	require.NotEmpty(t, u22.CreatedAt, "u2 created_at should not be empty")
	require.NotEmpty(t, u33.CreatedAt, "u3 created_at should not be empty")
	require.NotEmpty(t, u11.UpdatedAt, "u1 updated_at should not be empty")
	require.NotEmpty(t, u22.UpdatedAt, "u2 updated_at should not be empty")
	require.NotEmpty(t, u33.UpdatedAt, "u3 updated_at should not be empty")
	require.NotEmpty(t, u11.ID, "u1 id should not be empty")
	require.NotEmpty(t, u22.ID, "u2 id should not be empty")
	require.NotEmpty(t, u33.ID, "u3 id should not be empty")
	require.Equal(t, u1.Name, u11.Name, "u1 name should match")
	require.Equal(t, u2.Name, u22.Name, "u2 name should match")
	require.Equal(t, u3.Name, u33.Name, "u3 name should match")
	require.Equal(t, u1.Age, u11.Age, "u1 age should match")
	require.Equal(t, u2.Age, u22.Age, "u2 age should match")
	require.Equal(t, u3.Age, u33.Age, "u3 age should match")
	require.Equal(t, u1.Email, u11.Email, "u1 email should match")
	require.Equal(t, u2.Email, u22.Email, "u2 email should match")
	require.Equal(t, u3.Email, u33.Email, "u3 email should match")
	require.Equal(t, u1.IsActive, u11.IsActive, "u1 is_active should match")
	require.Equal(t, u2.IsActive, u22.IsActive, "u2 is_active should match")
	require.Equal(t, u3.IsActive, u33.IsActive, "u3 is_active should match")

	// Test Create with empty resources - should not return error
	require.NoError(t, database.Database[*TestUser](nil).Create(nil))
	require.NoError(t, database.Database[*TestUser](nil).Create([]*TestUser{nil, nil, nil}...))
	require.NoError(t, database.Database[*TestUser](nil).Create([]*TestUser{nil, u1, nil}...))
}

func TestDatabaseDelete(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic Delete - single record (soft delete)
	count := new(int64)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(3), *count, "should have 3 records initially")

	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(2), *count, "should have 2 records after soft delete")

	// Verify soft-deleted record is not visible in List
	users := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 2, len(users), "should have 2 records in List after soft delete")
	var foundU1 bool
	for _, u := range users {
		if u.ID == u1.ID {
			foundU1 = true
		}
	}
	require.False(t, foundU1, "u1 should not be found in List after soft delete")

	// Verify soft-deleted record is not accessible via Get
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.Empty(t, u.ID, "soft-deleted record should not be accessible via Get")

	// Test Delete - batch delete multiple records
	require.NoError(t, database.Database[*TestUser](nil).Delete(u2, u3))
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(0), *count, "should have 0 records after batch soft delete")

	// Verify all records are soft-deleted
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "should have 0 records in List after all soft deleted")

	// Recreate data for next test
	setupTestData(t)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(3), *count, "should have 3 records after recreate")

	// Test Delete - batch delete with slice
	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(0), *count, "should have 0 records after batch delete with slice")

	// Test Delete with empty resources - should not return error
	require.NoError(t, database.Database[*TestUser](nil).Delete(nil))
	require.NoError(t, database.Database[*TestUser](nil).Delete([]*TestUser{nil, nil, nil}...))
	require.NoError(t, database.Database[*TestUser](nil).Delete([]*TestUser{nil, u1, nil}...))
}

func TestDatabaseUpdate(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic Update - single record
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, u1.Name, u.Name)
	require.Equal(t, u1.Age, u.Age)
	require.Equal(t, u1.Email, u.Email)

	// Update single record
	u.Name = "user1_updated"
	u.Age = 25
	u.Email = "user1_updated@example.com"
	require.NoError(t, database.Database[*TestUser](nil).Update(u))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, "user1_updated", u.Name, "name should be updated")
	require.Equal(t, 25, u.Age, "age should be updated")
	require.Equal(t, "user1_updated@example.com", u.Email, "email should be updated")

	// Test Update - batch update multiple records
	u1.Name = "user1_batch"
	u2.Name = "user2_batch"
	u3.Name = "user3_batch"
	u1.Remark, u2.Remark, u3.Remark = nil, nil, nil // clear remark to test hook
	require.NoError(t, database.Database[*TestUser](nil).Update(ul...))
	count := new(int64)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(3), *count, "should have 3 records after batch update")

	// Check the update hook result
	require.Equal(t, remarkUserUpdateBefore, *u1.Remark, "u1 should have update hook result")
	require.Equal(t, remarkUserUpdateBefore, *u2.Remark, "u2 should have update hook result")
	require.Equal(t, remarkUserUpdateBefore, *u3.Remark, "u3 should have update hook result")

	// Verify updated data in the database
	users := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records")
	var u11, u22, u33 *TestUser
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			u11 = u
		case u2.ID:
			u22 = u
		case u3.ID:
			u33 = u
		}
	}
	require.NotNil(t, u11, "u1 should be found")
	require.NotNil(t, u22, "u2 should be found")
	require.NotNil(t, u33, "u3 should be found")
	require.NotEmpty(t, u11.CreatedAt, "created_at should not be empty")
	require.NotEmpty(t, u22.CreatedAt, "created_at should not be empty")
	require.NotEmpty(t, u33.CreatedAt, "created_at should not be empty")
	require.NotEmpty(t, u11.UpdatedAt, "updated_at should not be empty")
	require.NotEmpty(t, u22.UpdatedAt, "updated_at should not be empty")
	require.NotEmpty(t, u33.UpdatedAt, "updated_at should not be empty")
	require.NotEmpty(t, u11.ID, "id should not be empty")
	require.NotEmpty(t, u22.ID, "id should not be empty")
	require.NotEmpty(t, u33.ID, "id should not be empty")
	require.Equal(t, u1.Name, u11.Name, "u1 name should match")
	require.Equal(t, u2.Name, u22.Name, "u2 name should match")
	require.Equal(t, u3.Name, u33.Name, "u3 name should match")
	require.Equal(t, u1.Age, u11.Age, "u1 age should match")
	require.Equal(t, u2.Age, u22.Age, "u2 age should match")
	require.Equal(t, u3.Age, u33.Age, "u3 age should match")
	require.Equal(t, u1.Email, u11.Email, "u1 email should match")
	require.Equal(t, u2.Email, u22.Email, "u2 email should match")
	require.Equal(t, u3.Email, u33.Email, "u3 email should match")
	require.Equal(t, u1.IsActive, u11.IsActive, "u1 is_active should match")
	require.Equal(t, u2.IsActive, u22.IsActive, "u2 is_active should match")
	require.Equal(t, u3.IsActive, u33.IsActive, "u3 is_active should match")

	// Test Update with empty resources - should not return error
	require.NoError(t, database.Database[*TestUser](nil).Update(nil))
	require.NoError(t, database.Database[*TestUser](nil).Update([]*TestUser{nil, nil, nil}...))
	require.NoError(t, database.Database[*TestUser](nil).Update([]*TestUser{nil, u1, nil}...))
}

func TestDatabaseUpdateByID(t *testing.T) {
	defer cleanupTestData()

	require.NoError(t, database.Database[*TestUser](nil).Create(u1))
	// Test basic UpdateByID - update name field
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, u1.Name, u.Name)
	require.Equal(t, u1.Age, u.Age)
	require.Equal(t, u1.Email, u.Email)
	originalUpdatedAt := u.UpdatedAt

	newName := "user1_modified"
	require.NoError(t, database.Database[*TestUser](nil).UpdateByID(u.ID, "name", newName))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, newName, u.Name, "name should be updated")
	require.Equal(t, u1.Age, u.Age, "age should not be changed")
	require.Equal(t, u1.Email, u.Email, "email should not be changed")
	require.NotEqual(t, originalUpdatedAt, u.UpdatedAt, "updated_at should be updated")

	// Test UpdateByID - update age field
	newAge := 25
	previousUpdatedAt := u.UpdatedAt
	require.NoError(t, database.Database[*TestUser](nil).UpdateByID(u.ID, "age", newAge))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.Equal(t, newName, u.Name, "name should not be changed")
	require.Equal(t, newAge, u.Age, "age should be updated")
	require.Equal(t, u1.Email, u.Email, "email should not be changed")
	require.NotEqual(t, previousUpdatedAt, u.UpdatedAt, "updated_at should be updated again")

	// Test UpdateByID - update email field
	newEmail := "user1_new@example.com"
	previousUpdatedAt = u.UpdatedAt
	require.NoError(t, database.Database[*TestUser](nil).UpdateByID(u.ID, "email", newEmail))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.Equal(t, newName, u.Name, "name should not be changed")
	require.Equal(t, newAge, u.Age, "age should not be changed")
	require.Equal(t, newEmail, u.Email, "email should be updated")
	require.NotEqual(t, previousUpdatedAt, u.UpdatedAt, "updated_at should be updated again")

	// Test UpdateByID with non-existent ID - should not return error
	require.NoError(t, database.Database[*TestUser](nil).UpdateByID("non-existent-id", "name", "test"))

	// Test UpdateByID with empty parameters - should return errors
	err := database.Database[*TestUser](nil).UpdateByID("", "name", "value")
	require.Error(t, err, "should return error when id is empty")
	require.ErrorIs(t, err, database.ErrIDRequired, "error should be ErrIDRequired")

	err = database.Database[*TestUser](nil).UpdateByID("id", "", "value")
	require.Error(t, err, "should return error when name is empty")
	require.ErrorIs(t, err, database.ErrEmptyFieldName, "error should be ErrEmptyFieldName")

	err = database.Database[*TestUser](nil).UpdateByID("id", "name", nil)
	require.Error(t, err, "should return error when value is nil")
	require.ErrorIs(t, err, database.ErrNilValue, "error should be ErrNilValue")
}

func TestDatabaseList(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic List - should return all records
	users := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records")

	// Verify all records are returned correctly
	var u11, u22, u33 *TestUser
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			u11 = u
		case u2.ID:
			u22 = u
		case u3.ID:
			u33 = u
		}
	}
	require.NotNil(t, u11, "u1 should be found")
	require.NotNil(t, u22, "u2 should be found")
	require.NotNil(t, u33, "u3 should be found")
	require.NotEmpty(t, u11.CreatedAt)
	require.NotEmpty(t, u22.CreatedAt)
	require.NotEmpty(t, u33.CreatedAt)
	require.NotEmpty(t, u11.UpdatedAt)
	require.NotEmpty(t, u22.UpdatedAt)
	require.NotEmpty(t, u33.UpdatedAt)
	require.NotEmpty(t, u11.ID)
	require.NotEmpty(t, u22.ID)
	require.NotEmpty(t, u33.ID)
	require.Equal(t, u1.Name, u11.Name)
	require.Equal(t, u2.Name, u22.Name)
	require.Equal(t, u3.Name, u33.Name)
	require.Equal(t, u1.Age, u11.Age)
	require.Equal(t, u2.Age, u22.Age)
	require.Equal(t, u3.Age, u33.Age)
	require.Equal(t, u1.Email, u11.Email)
	require.Equal(t, u2.Email, u22.Email)
	require.Equal(t, u3.Email, u33.Email)
	require.Equal(t, u1.IsActive, u11.IsActive)
	require.Equal(t, u2.IsActive, u22.IsActive)
	require.Equal(t, u3.IsActive, u33.IsActive)

	// Test List with query conditions
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{Name: u1.Name}).List(&users))
	require.Equal(t, 1, len(users), "should have 1 record matching name")
	require.Equal(t, u1.Name, users[0].Name)

	// Test List after soft delete - should not return soft-deleted records
	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 2, len(users), "should have 2 records after soft delete")

	// Test List with empty result - should overwrite existing slice
	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
	users = make([]*TestUser, 0)
	users = append(users, u1, u2, u3) // Pre-populate with data
	require.Equal(t, 3, len(users), "slice should have 3 items before List")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "slice should be empty after List with no records")

	// Test List multiple times - should be idempotent
	require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users))
	users2 := make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users2))
	require.Equal(t, 3, len(users2))

	// Test List with different model types
	products := make([]*TestProduct, 0)
	require.NoError(t, database.Database[*TestProduct](nil).List(&products))
	require.GreaterOrEqual(t, len(products), 0, "product list should be non-negative")

	// Test List with nil dest - should return error
	err := database.Database[*TestUser](nil).List(nil)
	require.Error(t, err, "should return error when dest is nil")
	require.ErrorIs(t, err, database.ErrNilDest, "error should be ErrNilDest")
}

func TestDatabaseGet(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic Get - should return record by ID
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, u1.ID, u.ID, "should return u1 by ID")
	require.Equal(t, u1.Name, u.Name)
	require.Equal(t, u1.Age, u.Age)
	require.Equal(t, u1.Email, u.Email)

	// Test Get with different IDs
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u2.ID))
	require.Equal(t, u2.ID, u.ID, "should return u2 by ID")
	require.Equal(t, u2.Name, u.Name)
	require.Equal(t, u2.Age, u.Age)
	require.Equal(t, u2.Email, u.Email)

	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u3.ID))
	require.Equal(t, u3.ID, u.ID, "should return u3 by ID")
	require.Equal(t, u3.Name, u.Name)
	require.Equal(t, u3.Age, u.Age)
	require.Equal(t, u3.Email, u.Email)

	// Test Get with empty ID - should return error
	u = new(TestUser)
	err := database.Database[*TestUser](nil).Get(u, "")
	require.Error(t, err, "should return error when id is empty")
	require.ErrorIs(t, err, database.ErrIDRequired, "error should be ErrIDRequired")

	// Test Get with non-existent ID - should not return error
	u = new(TestUser)
	err = database.Database[*TestUser](nil).Get(u, "non-existent-id")
	require.NoError(t, err)
	require.Empty(t, u.ID)
	require.Empty(t, u.CreatedAt)
	require.Empty(t, u.UpdatedAt)
	require.Empty(t, u.Name)
	require.Empty(t, u.Age)
	require.Empty(t, u.Email)

	// Test Get after soft delete - should not return soft-deleted records
	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID)) // not returns error
	require.Empty(t, u.ID)
	require.Empty(t, u.CreatedAt)
	require.Empty(t, u.UpdatedAt)
	require.Empty(t, u.Name)
	require.Empty(t, u.Age)
	require.Empty(t, u.Email)

	// Test Get multiple times - should be idempotent
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u2.ID))
	require.Equal(t, u2.ID, u.ID)
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u2.ID))
	require.Equal(t, u2.ID, u.ID)

	// Test Get with different model types
	p := new(TestProduct)
	require.NoError(t, database.Database[*TestProduct](nil).Get(p, "non-existent-id"))
	require.Empty(t, p.ID)
	require.Empty(t, p.CreatedAt)
	require.Empty(t, p.UpdatedAt)

	// Test Get with nil dest - should returns error
	var uu *TestUser
	err = database.Database[*TestUser](nil).Get(uu, u1.ID)
	require.Error(t, err, "should return error when dest is nil")
	require.ErrorIs(t, err, database.ErrNilDest)
}

func TestDatabaseFirst(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic First - should return first record ordered by primary key
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).First(u))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, u1.Name, u.Name, "should return u1 (first record)")
	require.Equal(t, u1.Age, u.Age)
	require.Equal(t, u1.Email, u.Email)

	// Test First with query conditions
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{Name: u2.Name}).First(u))
	require.NotNil(t, u)
	require.Equal(t, u2.Name, u.Name, "should return u2 when querying by name")

	// Test First after soft delete - should not return soft-deleted records
	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).First(u))
	require.NotNil(t, u)
	require.Equal(t, u2.Name, u.Name, "should return u2 after u1 is soft-deleted")

	// Test First multiple times - should be idempotent
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).First(u))
	require.Equal(t, u2.Name, u.Name)
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).First(u))
	require.Equal(t, u2.Name, u.Name)

	// Test First with different model types
	p := new(TestProduct)
	err := database.Database[*TestProduct](nil).First(p)
	// First may return error if no records exist, which is acceptable
	if err != nil {
		require.Contains(t, err.Error(), "record not found", "should return 'record not found' error when no records exist")
	}

	// Test First with nil dest - should return error
	var nilFirst *TestUser
	err = database.Database[*TestUser](nil).First(nilFirst)
	require.Error(t, err, "should return error when dest is nil")
	require.ErrorIs(t, err, database.ErrNilDest)
}

func TestDatabaseLast(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic Last - should return last record ordered by primary key
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Last(u))
	require.NotNil(t, u)
	require.NotEmpty(t, u.CreatedAt)
	require.NotEmpty(t, u.UpdatedAt)
	require.NotEmpty(t, u.ID)
	require.Equal(t, u3.Name, u.Name, "should return u3 (last record)")
	require.Equal(t, u3.Age, u.Age)
	require.Equal(t, u3.Email, u.Email)

	// Test Last with query conditions
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{Name: u2.Name}).Last(u))
	require.NotNil(t, u)
	require.Equal(t, u2.Name, u.Name, "should return u2 when querying by name")

	// Test Last after soft delete - should not return soft-deleted records
	require.NoError(t, database.Database[*TestUser](nil).Delete(u3))
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Last(u))
	require.NotNil(t, u)
	require.Equal(t, u2.Name, u.Name, "should return u2 after u3 is soft-deleted")

	// Test Last multiple times - should be idempotent
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Last(u))
	require.Equal(t, u2.Name, u.Name)
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Last(u))
	require.Equal(t, u2.Name, u.Name)

	// Test Last with different model types
	p := new(TestProduct)
	err := database.Database[*TestProduct](nil).Last(p)
	// Last may return error if no records exist, which is acceptable
	if err != nil {
		require.Contains(t, err.Error(), "record not found", "should return 'record not found' error when no records exist")
	}

	// Test Last with nil dest - should return error
	var nilLast *TestUser
	err = database.Database[*TestUser](nil).Last(nilLast)
	require.Error(t, err, "should return error when dest is nil")
	require.ErrorIs(t, err, database.ErrNilDest)
}

func TestDatabaseTake(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test Take - should return a record
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Take(u))
	require.NotEmpty(t, u.ID)

	// Test Take with nil dest - should return error
	var nilTake *TestUser
	err := database.Database[*TestUser](nil).Take(nilTake)
	require.Error(t, err, "should return error when dest is nil")
	require.ErrorIs(t, err, database.ErrNilDest)
}

func TestDatabaseCount(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	// Test basic count - should return total number of records
	count := new(int64)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(3), *count, "should have 3 records")

	// Test count with query conditions
	require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{Name: u1.Name}).Count(count))
	require.Equal(t, int64(1), *count, "should have 1 record matching name")

	require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{Age: u2.Age}).Count(count))
	require.Equal(t, int64(1), *count, "should have 1 record matching age")

	// Test count after soft delete - soft-deleted records should not be counted
	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(2), *count, "should have 2 records after soft delete")

	// Test count with query after soft delete
	require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{Name: u1.Name}).Count(count))
	require.Equal(t, int64(0), *count, "soft-deleted record should not be counted")

	// Test count multiple times - should be idempotent
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(2), *count)
	require.NoError(t, database.Database[*TestUser](nil).Count(count))
	require.Equal(t, int64(2), *count)

	// Test count with different model types
	require.NoError(t, database.Database[*TestProduct](nil).Count(count))
	require.GreaterOrEqual(t, *count, int64(0), "product count should be non-negative")
	require.NoError(t, database.Database[*TestCategory](nil).Count(count))
	require.GreaterOrEqual(t, *count, int64(0), "category count should be non-negative")

	// Test count with nil pointer - should return error
	err := database.Database[*TestUser](nil).Count(nil)
	require.Error(t, err, "should return error when count is nil")
	require.Contains(t, err.Error(), "count parameter cannot be nil", "error message should indicate nil pointer issue")
}

func TestDatabaseTransaction(t *testing.T) {
	defer cleanupTestData()

	flag := 0
	users := make([]*TestUser, 0)

	// Test Transaction - transaction success
	// Transaction automatically injects txDB, no need for WithTx
	err := database.Database[*TestUser](nil).Transaction(func(txDB types.Database[*TestUser]) error {
		// No need to call WithTx - txDB already has transaction context
		return txDB.Create(ul...)
	})
	require.NoError(t, err, "transaction should succeed")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records after successful transaction")

	// Verify created data integrity
	var foundU1, foundU2, foundU3 bool
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			foundU1 = true
			require.Equal(t, u1.Name, u.Name, "u1 name should match")
		case u2.ID:
			foundU2 = true
			require.Equal(t, u2.Name, u.Name, "u2 name should match")
		case u3.ID:
			foundU3 = true
			require.Equal(t, u3.Name, u.Name, "u3 name should match")
		}
	}
	require.True(t, foundU1 && foundU2 && foundU3, "all users should be found")

	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

	// Test Transaction - transaction failed with rollback
	// Rollback will execute if transaction failed, so resources will not be created
	err = database.Database[*TestUser](nil).Transaction(func(txDB types.Database[*TestUser]) error {
		require.NoError(t, txDB.Create(ul...))
		return errors.New("test error")
	})
	require.Error(t, err, "transaction should fail")
	require.Contains(t, err.Error(), "test error", "error should contain test error message")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "should have 0 records after rollback")

	// Test Transaction - multiple operations in transaction
	err = database.Database[*TestUser](nil).Transaction(func(txDB types.Database[*TestUser]) error {
		// Create users
		if txErr := txDB.Create(u1); txErr != nil {
			return txErr
		}
		// Update user in the same transaction
		u1.Name = "user1_updated"
		if txErr := txDB.Update(u1); txErr != nil {
			return txErr
		}
		// UpdateByID in the same transaction
		return txDB.UpdateByID(u1.ID, "age", 25)
	})
	require.NoError(t, err, "transaction should succeed")

	// Verify the updates were committed
	u := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.Equal(t, "user1_updated", u.Name, "name should be updated")
	require.Equal(t, 25, u.Age, "age should be updated")

	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))

	// Test Transaction - transaction success with custom rollback function
	// Rollback function should not execute if transaction succeeds
	err = database.Database[*TestUser](nil).WithRollback(func() {
		flag++
	}).Transaction(func(txDB types.Database[*TestUser]) error {
		return txDB.Create(ul...)
	})
	require.NoError(t, err, "transaction should succeed")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records after successful transaction")
	require.Equal(t, 0, flag, "rollback function should not be called on success")

	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

	// Test Transaction - transaction failed with custom rollback function
	// Rollback function should execute if transaction fails
	err = database.Database[*TestUser](nil).WithRollback(func() {
		flag++
	}).Transaction(func(txDB types.Database[*TestUser]) error {
		require.NoError(t, txDB.Create(ul...))
		return errors.New("test error")
	})
	require.Error(t, err, "transaction should fail")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "should have 0 records after rollback")
	require.Equal(t, 1, flag, "rollback function should be called on failure")

	// Test Transaction - with query options (WithLock, WithQuery, etc.)
	flag = 0
	require.NoError(t, database.Database[*TestUser](nil).Create(u1))
	err = database.Database[*TestUser](nil).Transaction(func(txDB types.Database[*TestUser]) error {
		lockedUser := new(TestUser)
		// Test WithLock works in transaction
		if lockErr := txDB.WithLock(consts.LockUpdate).Get(lockedUser, u1.ID); lockErr != nil {
			return lockErr
		}
		lockedUser.Name = "locked_update"
		return txDB.Update(lockedUser)
	})
	require.NoError(t, err, "transaction with lock should succeed")
	u = new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
	require.Equal(t, "locked_update", u.Name, "name should be updated")

	require.NoError(t, database.Database[*TestUser](nil).Delete(u1))
}

func TestDatabaseTransactionFunc(t *testing.T) {
	defer cleanupTestData()

	flag := 0
	users := make([]*TestUser, 0)

	// Test TransactionFunc - transaction success
	err := database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
		return nil
	})
	require.NoError(t, err, "transaction should succeed")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records after successful transaction")
	require.Equal(t, 0, flag, "rollback function should not be called on success")

	// Verify created data integrity
	var foundU1, foundU2, foundU3 bool
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			foundU1 = true
			require.Equal(t, u1.Name, u.Name, "u1 name should match")
		case u2.ID:
			foundU2 = true
			require.Equal(t, u2.Name, u.Name, "u2 name should match")
		case u3.ID:
			foundU3 = true
			require.Equal(t, u3.Name, u.Name, "u3 name should match")
		}
	}
	require.True(t, foundU1 && foundU2 && foundU3, "all users should be found")

	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

	// Test TransactionFunc - transaction failed with rollback
	// Rollback will execute if transaction failed, so resource will not be created
	err = database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
		return errors.New("test error")
	})
	require.Error(t, err, "transaction should fail")
	require.Contains(t, err.Error(), "test error", "error should contain test error message")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "should have 0 records after rollback")
	require.Equal(t, 0, flag, "rollback function should not be called without WithRollback")

	// Test TransactionFunc - incorrect use (not using WithTx)
	// Rollback will not execute if not using WithTx option, so resources will be created outside transaction
	err = database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		return errors.New("test error")
	})
	require.Error(t, err, "transaction should fail")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records because Create was not in transaction")
	require.Equal(t, 0, flag, "rollback function should not be called")

	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

	// Test TransactionFunc - transaction success with custom rollback function
	// Rollback function should not execute if transaction succeeds
	err = database.Database[*TestUser](nil).WithRollback(func() {
		flag++
	}).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
		return nil
	})
	require.NoError(t, err, "transaction should succeed")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 3, len(users), "should have 3 records after successful transaction")
	require.Equal(t, 0, flag, "rollback function should not be called on success")

	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

	// Test TransactionFunc - transaction failed with custom rollback function
	// Rollback function should execute if transaction fails
	err = database.Database[*TestUser](nil).WithRollback(func() {
		flag++
	}).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
		return errors.New("test error")
	})
	require.Error(t, err, "transaction should fail")
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "should have 0 records after rollback")
	require.Equal(t, 1, flag, "rollback function should be called on failure")
}

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

func TestDatabaseWithTx(t *testing.T) {
	defer func() {
		cleanupTestData()
		_ = database.Database[*TestProduct](nil).Delete()
	}()

	// Transaction success - Create operation
	err := database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
		return nil
	})
	require.NoError(t, err)
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
	require.True(t, foundU1 && foundU2 && foundU3, "all users should be found")

	// Transaction success - Update operation
	require.NoError(t, database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		u1.Name = "user1_updated"
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Update(u1))
		return nil
	}))
	user := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).Get(user, u1.ID))
	require.Equal(t, "user1_updated", user.Name)
	u1.Name = "user1" // restore

	// Transaction success - Multiple resource types
	p1 := &TestProduct{Name: "product1", Price: 10.0, Base: model.Base{ID: "p1"}}
	require.NoError(t, database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(u1))
		require.NoError(t, database.Database[*TestProduct](nil).WithTx(tx).Create(p1))
		return nil
	}))
	product := new(TestProduct)
	require.NoError(t, database.Database[*TestProduct](nil).Get(product, p1.ID))
	require.NotNil(t, product)
	require.Equal(t, p1.Name, product.Name)

	// Transaction success - List operation within transaction
	require.NoError(t, database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		txUsers := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).List(&txUsers))
		require.Greater(t, len(txUsers), 0, "should find users within transaction")
		return nil
	}))

	// Transaction failed - rollback on error
	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
	err = database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
		return errors.New("custom error")
	})
	require.Error(t, err)
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.Equal(t, 0, len(users), "transaction should be rolled back, no users created")

	// Transaction with chainable methods
	require.NoError(t, database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
		require.NoError(t, database.Database[*TestUser](nil).
			WithTx(tx).
			WithQuery(&TestUser{Name: u1.Name}).
			Create(u1))
		return nil
	}))
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).List(&users))
	require.GreaterOrEqual(t, len(users), 1, "should find created user")
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

func TestDatabaseWithIndex(t *testing.T) {
	defer cleanupTestData()
	setupTestData(t)

	existsIndex := "idx_test_users_created_by" // index auto created when database migration.
	notExistsIndex := "not_exists_index"

	users := make([]*TestUser, 0)

	// Test WithIndex with default hint (USE INDEX)
	// Note: Index hints only work on MySQL. On SQLite/PostgreSQL, it will skip silently.
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex).List(&users))
	require.Equal(t, 3, len(users))
	// Verify returned data integrity
	var foundU1, foundU2, foundU3 bool
	for _, u := range users {
		switch u.ID {
		case u1.ID:
			foundU1 = true
			require.NotEmpty(t, u.ID)
			require.NotEmpty(t, u.CreatedAt)
			require.NotEmpty(t, u.UpdatedAt)
			require.Equal(t, u1.Name, u.Name)
			require.Equal(t, u1.Age, u.Age)
			require.Equal(t, u1.Email, u.Email)
		case u2.ID:
			foundU2 = true
			require.NotEmpty(t, u.ID)
			require.NotEmpty(t, u.CreatedAt)
			require.NotEmpty(t, u.UpdatedAt)
			require.Equal(t, u2.Name, u.Name)
			require.Equal(t, u2.Age, u.Age)
			require.Equal(t, u2.Email, u.Email)
		case u3.ID:
			foundU3 = true
			require.NotEmpty(t, u.ID)
			require.NotEmpty(t, u.CreatedAt)
			require.NotEmpty(t, u.UpdatedAt)
			require.Equal(t, u3.Name, u.Name)
			require.Equal(t, u3.Age, u.Age)
			require.Equal(t, u3.Email, u.Email)
		}
	}
	require.True(t, foundU1, "should find u1")
	require.True(t, foundU2, "should find u2")
	require.True(t, foundU3, "should find u3")

	// Test WithIndex with explicit USE hint
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex, consts.IndexHintUse).List(&users))
	require.Equal(t, 3, len(users))

	// Test WithIndex with FORCE hint
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex, consts.IndexHintForce).List(&users))
	require.Equal(t, 3, len(users))

	// Test WithIndex with IGNORE hint
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex, consts.IndexHintIgnore).List(&users))
	require.Equal(t, 3, len(users))

	// Test WithIndex with empty index name (should be ignored)
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex("").List(&users))
	require.Equal(t, 3, len(users), "empty index name should be ignored and query should work normally")

	// Test WithIndex with whitespace-only index name (should be ignored)
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex("   ").List(&users))
	require.Equal(t, 3, len(users), "whitespace-only index name should be ignored and query should work normally")

	// Test WithIndex combined with WithQuery
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).
		WithIndex(existsIndex).
		WithQuery(&TestUser{Name: u1.Name}).
		List(&users))
	require.Equal(t, 1, len(users))
	require.Equal(t, u1.ID, users[0].ID)
	require.Equal(t, u1.Name, users[0].Name)

	// Test WithIndex with non-existent index
	// Note: On MySQL, this will cause an error. On SQLite/PostgreSQL, it will skip silently.
	// The behavior depends on the database type, so we test that it doesn't panic.
	users = make([]*TestUser, 0)
	// On SQLite, this will skip the index hint and work normally
	// On MySQL, this might cause an error depending on the index existence
	err := database.Database[*TestUser](nil).WithIndex(notExistsIndex).List(&users)
	// We don't assert error here because behavior differs by database type
	_ = err

	// Test WithIndex with empty result set
	require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
	users = make([]*TestUser, 0)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex).List(&users))
	require.Equal(t, 0, len(users), "should return empty result when no records exist")

	// Test WithIndex with Get method (single record)
	require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
	user := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex).Get(user, u1.ID))
	require.NotNil(t, user)
	require.Equal(t, u1.ID, user.ID)
	require.Equal(t, u1.Name, user.Name)
	require.Equal(t, u1.Age, user.Age)
	require.Equal(t, u1.Email, user.Email)

	// Test WithIndex with Count method
	count := new(int64)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex).Count(count))
	require.Equal(t, int64(3), *count)

	// Test WithIndex with First method
	firstUser := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex).First(firstUser))
	require.NotNil(t, firstUser)
	require.NotEmpty(t, firstUser.ID)

	// Test WithIndex with Last method
	lastUser := new(TestUser)
	require.NoError(t, database.Database[*TestUser](nil).WithIndex(existsIndex).Last(lastUser))
	require.NotNil(t, lastUser)
	require.NotEmpty(t, lastUser.ID)
}

func TestDatabaseWithQuery(t *testing.T) {
	t.Run("ExactMatch", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		users := make([]*TestUser, 0)

		// Test exact match by Name field: query each user by name
		testCases := []struct {
			name     string
			query    *TestUser
			expected *TestUser
		}{
			{"query u1 by name", &TestUser{Name: u1.Name}, u1},
			{"query u2 by name", &TestUser{Name: u2.Name}, u2},
			{"query u3 by name", &TestUser{Name: u3.Name}, u3},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				users = make([]*TestUser, 0)
				require.NoError(t, database.Database[*TestUser](nil).
					WithQuery(tc.query).
					List(&users))
				require.Equal(t, 1, len(users))
				u := users[0]
				require.NotNil(t, u)
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, tc.expected.ID, u.ID)
				require.Equal(t, tc.expected.Name, u.Name)
				require.Equal(t, tc.expected.Age, u.Age)
				require.Equal(t, tc.expected.Email, u.Email)
				require.Equal(t, tc.expected.IsActive, u.IsActive)
			})
		}

		// Test exact match by Age field: query each user by age
		ageTestCases := []struct {
			name     string
			query    *TestUser
			expected *TestUser
		}{
			{"query u1 by age", &TestUser{Age: u1.Age}, u1},
			{"query u2 by age", &TestUser{Age: u2.Age}, u2},
			{"query u3 by age", &TestUser{Age: u3.Age}, u3},
		}

		for _, tc := range ageTestCases {
			t.Run(tc.name, func(t *testing.T) {
				users = make([]*TestUser, 0)
				require.NoError(t, database.Database[*TestUser](nil).
					WithQuery(tc.query).
					List(&users))
				require.Equal(t, 1, len(users))
				u := users[0]
				require.NotNil(t, u)
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, tc.expected.ID, u.ID)
				require.Equal(t, tc.expected.Name, u.Name)
				require.Equal(t, tc.expected.Age, u.Age)
				require.Equal(t, tc.expected.Email, u.Email)
				require.Equal(t, tc.expected.IsActive, u.IsActive)
			})
		}

		// Test exact match by Email field: query each user by email
		emailTestCases := []struct {
			name     string
			query    *TestUser
			expected *TestUser
		}{
			{"query u1 by email", &TestUser{Email: u1.Email}, u1},
			{"query u2 by email", &TestUser{Email: u2.Email}, u2},
			{"query u3 by email", &TestUser{Email: u3.Email}, u3},
		}

		for _, tc := range emailTestCases {
			t.Run(tc.name, func(t *testing.T) {
				users = make([]*TestUser, 0)
				require.NoError(t, database.Database[*TestUser](nil).
					WithQuery(tc.query).
					List(&users))
				require.Equal(t, 1, len(users))
				u := users[0]
				require.NotNil(t, u)
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, tc.expected.ID, u.ID)
				require.Equal(t, tc.expected.Name, u.Name)
				require.Equal(t, tc.expected.Age, u.Age)
				require.Equal(t, tc.expected.Email, u.Email)
				require.Equal(t, tc.expected.IsActive, u.IsActive)
			})
		}

		// Test exact match with multiple fields (AND logic): Name and Age
		// Query: Name="user1" AND Age=18 should return u1
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Age: u1.Age}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)
		require.Equal(t, u1.Age, users[0].Age)
		require.Equal(t, u1.Email, users[0].Email)

		// Test exact match with multiple fields that don't match: Name="user1" AND Age=19
		// Should return 0 records (no user matches both)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Age: u2.Age}).
			List(&users))
		require.Equal(t, 0, len(users), "multiple fields with AND logic should match all conditions")

		// Test exact match with three fields: Name, Age, and Email
		// Query: Name="user1" AND Age=18 AND Email="user1@example.com" should return u1
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Age: u1.Age, Email: u1.Email}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)
		require.Equal(t, u1.Age, users[0].Age)
		require.Equal(t, u1.Email, users[0].Email)

		// Test exact match with non-existent value: should return 0 records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "nonexistent"}).
			List(&users))
		require.Equal(t, 0, len(users), "non-existent value should return 0 records")

		// Test exact match with non-existent age: should return 0 records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Age: 999}).
			List(&users))
		require.Equal(t, 0, len(users), "non-existent age should return 0 records")
	})

	t.Run("MultipleValues", func(t *testing.T) {
		t.Run("multiple_id", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			users := make([]*TestUser, 0)

			// Test multiple IDs with comma-separated values: ID="u1,u2"
			// Should return 2 records (u1 and u2) using IN clause
			query := new(TestUser)
			ids := []string{u1.ID, u2.ID}
			query.ID = strings.Join(ids, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 2, len(users))

			var u11, u22 *TestUser
			for _, u := range users {
				switch u.ID {
				case u1.ID:
					u11 = u
				case u2.ID:
					u22 = u
				}
			}
			require.NotNil(t, u11, "should find u1")
			require.NotNil(t, u22, "should find u2")
			require.NotEmpty(t, u11.ID)
			require.NotEmpty(t, u22.ID)
			require.NotEmpty(t, u11.CreatedAt)
			require.NotEmpty(t, u22.CreatedAt)
			require.NotEmpty(t, u11.UpdatedAt)
			require.NotEmpty(t, u22.UpdatedAt)
			require.Equal(t, u1.Name, u11.Name)
			require.Equal(t, u2.Name, u22.Name)
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u1.Email, u11.Email)
			require.Equal(t, u2.Email, u22.Email)
			require.Equal(t, u1.IsActive, u11.IsActive)
			require.Equal(t, u2.IsActive, u22.IsActive)

			// Test multiple IDs with three values: ID="u1,u2,u3"
			// Should return all 3 records
			users = make([]*TestUser, 0)
			query = new(TestUser)
			ids = []string{u1.ID, u2.ID, u3.ID}
			query.ID = strings.Join(ids, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 3, len(users))
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
					require.Equal(t, u2.Age, u.Age)
					require.Equal(t, u2.Email, u.Email)
				case u3.ID:
					foundU3 = true
					require.Equal(t, u3.Name, u.Name)
					require.Equal(t, u3.Age, u.Age)
					require.Equal(t, u3.Email, u.Email)
				}
			}
			require.True(t, foundU1, "should find u1")
			require.True(t, foundU2, "should find u2")
			require.True(t, foundU3, "should find u3")

			// Test multiple IDs with non-existent ID: ID="u1,nonexistent"
			// Should return only u1 (non-existent ID is ignored)
			users = make([]*TestUser, 0)
			query = new(TestUser)
			ids = []string{u1.ID, "nonexistent-id"}
			query.ID = strings.Join(ids, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 1, len(users))
			require.Equal(t, u1.ID, users[0].ID)
			require.Equal(t, u1.Name, users[0].Name)

			// Test multiple IDs with single value: ID="u1"
			// Should return 1 record (single value should work)
			users = make([]*TestUser, 0)
			query = new(TestUser)
			query.ID = u1.ID
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 1, len(users))
			require.Equal(t, u1.ID, users[0].ID)
			require.Equal(t, u1.Name, users[0].Name)
		})

		t.Run("multiple_name", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			users := make([]*TestUser, 0)

			// Test multiple names with comma-separated values: Name="user2,user3"
			// Should return 2 records (u2 and u3) using IN clause
			query := new(TestUser)
			names := []string{u2.Name, u3.Name}
			query.Name = strings.Join(names, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 2, len(users))

			var u22, u33 *TestUser
			for _, u := range users {
				switch u.ID {
				case u2.ID:
					u22 = u
				case u3.ID:
					u33 = u
				}
			}
			require.NotNil(t, u22, "should find u2")
			require.NotNil(t, u33, "should find u3")
			require.NotEmpty(t, u22.ID)
			require.NotEmpty(t, u33.ID)
			require.NotEmpty(t, u22.CreatedAt)
			require.NotEmpty(t, u33.CreatedAt)
			require.NotEmpty(t, u22.UpdatedAt)
			require.NotEmpty(t, u33.UpdatedAt)
			require.Equal(t, u2.Name, u22.Name)
			require.Equal(t, u3.Name, u33.Name)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u3.Age, u33.Age)
			require.Equal(t, u2.Email, u22.Email)
			require.Equal(t, u3.Email, u33.Email)
			require.Equal(t, u2.IsActive, u22.IsActive)
			require.Equal(t, u3.IsActive, u33.IsActive)

			// Test multiple names with three values: Name="user1,user2,user3"
			// Should return all 3 records
			users = make([]*TestUser, 0)
			query = new(TestUser)
			names = []string{u1.Name, u2.Name, u3.Name}
			query.Name = strings.Join(names, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 3, len(users))
			var foundU1, foundU2, foundU3 bool
			for _, u := range users {
				switch u.ID {
				case u1.ID:
					foundU1 = true
					require.Equal(t, u1.Name, u.Name)
					require.Equal(t, u1.Age, u.Age)
				case u2.ID:
					foundU2 = true
					require.Equal(t, u2.Name, u.Name)
					require.Equal(t, u2.Age, u.Age)
				case u3.ID:
					foundU3 = true
					require.Equal(t, u3.Name, u.Name)
					require.Equal(t, u3.Age, u.Age)
				}
			}
			require.True(t, foundU1, "should find u1")
			require.True(t, foundU2, "should find u2")
			require.True(t, foundU3, "should find u3")

			// Test multiple names with non-existent name: Name="user1,nonexistent"
			// Should return only u1 (non-existent name is ignored)
			users = make([]*TestUser, 0)
			query = new(TestUser)
			names = []string{u1.Name, "nonexistent"}
			query.Name = strings.Join(names, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 1, len(users))
			require.Equal(t, u1.ID, users[0].ID)
			require.Equal(t, u1.Name, users[0].Name)

			// Test multiple names with single value: Name="user1"
			// Should return 1 record (single value should work)
			users = make([]*TestUser, 0)
			query = new(TestUser)
			query.Name = u1.Name
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 1, len(users))
			require.Equal(t, u1.ID, users[0].ID)
			require.Equal(t, u1.Name, users[0].Name)
		})

		t.Run("multiple_email", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			users := make([]*TestUser, 0)

			// Test multiple emails with comma-separated values: Email="user1@example.com,user2@example.com"
			// Should return 2 records (u1 and u2) using IN clause
			query := new(TestUser)
			emails := []string{u1.Email, u2.Email}
			query.Email = strings.Join(emails, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 2, len(users))

			var u11, u22 *TestUser
			for _, u := range users {
				switch u.ID {
				case u1.ID:
					u11 = u
				case u2.ID:
					u22 = u
				}
			}
			require.NotNil(t, u11, "should find u1")
			require.NotNil(t, u22, "should find u2")
			require.NotEmpty(t, u11.ID)
			require.NotEmpty(t, u22.ID)
			require.NotEmpty(t, u11.CreatedAt)
			require.NotEmpty(t, u22.CreatedAt)
			require.NotEmpty(t, u11.UpdatedAt)
			require.NotEmpty(t, u22.UpdatedAt)
			require.Equal(t, u1.Name, u11.Name)
			require.Equal(t, u2.Name, u22.Name)
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u1.Email, u11.Email)
			require.Equal(t, u2.Email, u22.Email)
			require.Equal(t, u1.IsActive, u11.IsActive)
			require.Equal(t, u2.IsActive, u22.IsActive)

			// Test multiple emails with three values
			users = make([]*TestUser, 0)
			query = new(TestUser)
			emails = []string{u1.Email, u2.Email, u3.Email}
			query.Email = strings.Join(emails, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 3, len(users))
			var foundU1, foundU2, foundU3 bool
			for _, u := range users {
				switch u.ID {
				case u1.ID:
					foundU1 = true
				case u2.ID:
					foundU2 = true
				case u3.ID:
					foundU3 = true
				}
			}
			require.True(t, foundU1, "should find u1")
			require.True(t, foundU2, "should find u2")
			require.True(t, foundU3, "should find u3")
		})

		t.Run("multiple_fields", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			users := make([]*TestUser, 0)

			// Test multiple fields with comma-separated values: Name="user1,user2" AND Email="user1@example.com,user2@example.com"
			// Should return 2 records (u1 and u2) - both fields use IN clause with AND logic
			query := new(TestUser)
			names := []string{u1.Name, u2.Name}
			emails := []string{u1.Email, u2.Email}
			query.Name = strings.Join(names, ",")
			query.Email = strings.Join(emails, ",")
			require.NoError(t, database.Database[*TestUser](nil).WithQuery(query).List(&users))
			require.Equal(t, 2, len(users))
			var foundU1, foundU2 bool
			for _, u := range users {
				switch u.ID {
				case u1.ID:
					foundU1 = true
					require.Equal(t, u1.Name, u.Name)
					require.Equal(t, u1.Email, u.Email)
				case u2.ID:
					foundU2 = true
					require.Equal(t, u2.Name, u.Name)
					require.Equal(t, u2.Email, u.Email)
				}
			}
			require.True(t, foundU1, "should find u1")
			require.True(t, foundU2, "should find u2")
		})
	})

	t.Run("FuzzyMatch", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		users := make([]*TestUser, 0)

		// Test FuzzyMatch=false (default, exact match): should return 0 records for partial match
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user"}, types.QueryConfig{
				FuzzyMatch: false,
			}).
			List(&users))
		require.Equal(t, 0, len(users), "FuzzyMatch=false should not match partial strings")

		// Test FuzzyMatch=true with single value (LIKE): query "name" with partial match
		// Should return all 3 records (user1, user2, user3 all contain "user")
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users))
		var u11, u22, u33 *TestUser
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			case u3.ID:
				u33 = u
			}
		}
		require.NotNil(t, u11)
		require.NotNil(t, u22)
		require.NotNil(t, u33)
		require.NotEmpty(t, u11.ID)
		require.NotEmpty(t, u22.ID)
		require.NotEmpty(t, u33.ID)
		require.NotEmpty(t, u11.CreatedAt)
		require.NotEmpty(t, u22.CreatedAt)
		require.NotEmpty(t, u33.CreatedAt)
		require.NotEmpty(t, u11.UpdatedAt)
		require.NotEmpty(t, u22.UpdatedAt)
		require.NotEmpty(t, u33.UpdatedAt)
		require.Equal(t, u1.Name, u11.Name)
		require.Equal(t, u2.Name, u22.Name)
		require.Equal(t, u3.Name, u33.Name)
		require.Equal(t, u1.Age, u11.Age)
		require.Equal(t, u2.Age, u22.Age)
		require.Equal(t, u3.Age, u33.Age)
		require.Equal(t, u1.Email, u11.Email)
		require.Equal(t, u2.Email, u22.Email)
		require.Equal(t, u3.Email, u33.Email)
		require.Equal(t, u1.IsActive, u11.IsActive)
		require.Equal(t, u2.IsActive, u22.IsActive)
		require.Equal(t, u3.IsActive, u33.IsActive)

		// Test FuzzyMatch=true with single value (LIKE): query "email" with partial match
		// Should return all 3 records (all emails contain "example.com")
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Email: "example.com"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users))
		u11, u22, u33 = nil, nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			case u3.ID:
				u33 = u
			}
		}
		require.NotNil(t, u11)
		require.NotNil(t, u22)
		require.NotNil(t, u33)
		require.NotEmpty(t, u11.ID)
		require.NotEmpty(t, u22.ID)
		require.NotEmpty(t, u33.ID)
		require.NotEmpty(t, u11.CreatedAt)
		require.NotEmpty(t, u22.CreatedAt)
		require.NotEmpty(t, u33.CreatedAt)
		require.NotEmpty(t, u11.UpdatedAt)
		require.NotEmpty(t, u22.UpdatedAt)
		require.NotEmpty(t, u33.UpdatedAt)
		require.Equal(t, u1.Name, u11.Name)
		require.Equal(t, u2.Name, u22.Name)
		require.Equal(t, u3.Name, u33.Name)
		require.Equal(t, u1.Age, u11.Age)
		require.Equal(t, u2.Age, u22.Age)
		require.Equal(t, u3.Age, u33.Age)
		require.Equal(t, u1.Email, u11.Email)
		require.Equal(t, u2.Email, u22.Email)
		require.Equal(t, u3.Email, u33.Email)
		require.Equal(t, u1.IsActive, u11.IsActive)
		require.Equal(t, u2.IsActive, u22.IsActive)
		require.Equal(t, u3.IsActive, u33.IsActive)

		// Test FuzzyMatch=true with single value (LIKE): exact match should still work
		// Query: Name="user1" should return 1 record (u1)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user1"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		u := users[0]
		require.NotNil(t, u)
		require.NotEmpty(t, u.CreatedAt)
		require.NotEmpty(t, u.UpdatedAt)
		require.Equal(t, u1.ID, u.ID)
		require.Equal(t, u1.Name, u.Name)
		require.Equal(t, u1.Age, u.Age)
		require.Equal(t, u1.Email, u.Email)
		require.Equal(t, u1.IsActive, u.IsActive)

		// Test FuzzyMatch=true with multiple values (REGEXP): comma-separated values
		// Query: Name="user1,user2" should return 2 records (u1 and u2)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: strings.Join([]string{u1.Name, u2.Name}, ",")}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 2, len(users))
		u11, u22 = nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			}
		}
		require.NotNil(t, u11, "should find u1")
		require.NotNil(t, u22, "should find u2")
		require.Equal(t, u1.Name, u11.Name)
		require.Equal(t, u2.Name, u22.Name)
		require.Equal(t, u1.Age, u11.Age)
		require.Equal(t, u2.Age, u22.Age)
		require.Equal(t, u1.Email, u11.Email)
		require.Equal(t, u2.Email, u22.Email)

		// Test FuzzyMatch=true with multiple values (REGEXP): partial matches in comma-separated values
		// Query: Name="user,ser" should return all 3 records (all contain "user")
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user,ser"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users))
		u11, u22, u33 = nil, nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			case u3.ID:
				u33 = u
			}
		}
		require.NotNil(t, u11, "should find u1")
		require.NotNil(t, u22, "should find u2")
		require.NotNil(t, u33, "should find u3")

		// Test FuzzyMatch=true with no matching value: should return 0 records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "nonexistent"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 0, len(users))

		// Test FuzzyMatch=true with empty string: should return 0 records (empty query blocked)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: ""}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 0, len(users), "empty query should be blocked by default")

		// Test FuzzyMatch=true with multiple fields: Name and Email
		// Query: Name="user" AND Email="example" should return all 3 records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user", Email: "example"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users))
		u11, u22, u33 = nil, nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			case u3.ID:
				u33 = u
			}
		}
		require.NotNil(t, u11, "should find u1")
		require.NotNil(t, u22, "should find u2")
		require.NotNil(t, u33, "should find u3")

		// Test FuzzyMatch=true with comma-separated values containing empty strings
		// Query: Name="user1,,user2" should return 2 records (u1 and u2), empty strings should be ignored
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user1,,user2"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 2, len(users), "empty strings in comma-separated values should be ignored")
		u11, u22 = nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			}
		}
		require.NotNil(t, u11, "should find u1")
		require.NotNil(t, u22, "should find u2")
		require.Equal(t, u1.Name, u11.Name)
		require.Equal(t, u2.Name, u22.Name)
		require.Equal(t, u1.Age, u11.Age)
		require.Equal(t, u2.Age, u22.Age)
		require.Equal(t, u1.Email, u11.Email)
		require.Equal(t, u2.Email, u22.Email)

		// Test FuzzyMatch=true with partial match at start: Name="1" (matches "user1")
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "1"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 1, len(users), "should match partial string at end")
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)

		// Test FuzzyMatch=true with partial match in middle: Name="ser" (matches all users)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "ser"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "should match partial string in middle")
		foundU1, foundU2, foundU3 := false, false, false
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1 = true
			case u2.ID:
				foundU2 = true
			case u3.ID:
				foundU3 = true
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test FuzzyMatch=true with partial match at end: Name="user" (matches all users)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "should match partial string at start")

		// Test FuzzyMatch=true with email partial match: Email="@example" (matches all emails)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Email: "@example"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "should match email partial string")
		foundU1, foundU2, foundU3 = false, false, false
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1 = true
			case u2.ID:
				foundU2 = true
			case u3.ID:
				foundU3 = true
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test FuzzyMatch=true with REGEXP special characters (should be escaped)
		// Query: Name="user1,user2" with special regex chars should work correctly
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: strings.Join([]string{u1.Name, u2.Name}, ",")}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 2, len(users), "REGEXP special characters should be escaped")
		u11, u22 = nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u
			}
		}
		require.NotNil(t, u11, "should find u1")
		require.NotNil(t, u22, "should find u2")

		// Test FuzzyMatch=true with multiple comma-separated values: Name="user1,user3"
		// Should return 2 records (u1 and u3)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: strings.Join([]string{u1.Name, u3.Name}, ",")}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 2, len(users))
		u11, u33 = nil, nil
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u3.ID:
				u33 = u
			}
		}
		require.NotNil(t, u11, "should find u1")
		require.NotNil(t, u33, "should find u3")
		require.Equal(t, u1.Name, u11.Name)
		require.Equal(t, u3.Name, u33.Name)
		require.Equal(t, u1.Age, u11.Age)
		require.Equal(t, u3.Age, u33.Age)

		// Test FuzzyMatch=true with AllowEmpty: empty query should return all records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{}, types.QueryConfig{
				FuzzyMatch: true,
				AllowEmpty: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "FuzzyMatch with AllowEmpty should return all records")

		// Test FuzzyMatch=true with UseOr: Name="user1" OR Email="user2@example.com"
		// Should return u1 (matches Name) and u2 (matches Email)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Email: u2.Email}, types.QueryConfig{
				FuzzyMatch: true,
				UseOr:      true,
			}).
			List(&users))
		require.Equal(t, 2, len(users), "FuzzyMatch with UseOr should work correctly")
		foundU1, foundU2 = false, false
		for _, u := range users {
			if u.ID == u1.ID {
				foundU1 = true
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Email, u.Email)
			}
			if u.ID == u2.ID {
				foundU2 = true
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Email, u.Email)
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")

		// Test FuzzyMatch=true with single field and empty string value (should be blocked)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "", Email: "example"}, types.QueryConfig{
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "query with some non-empty fields should work even with empty strings")

		// Test FuzzyMatch=false explicitly (should be same as default)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user"}, types.QueryConfig{
				FuzzyMatch: false,
			}).
			List(&users))
		require.Equal(t, 0, len(users), "FuzzyMatch=false should not match partial strings")
	})

	t.Run("AllowEmpty", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		users := make([]*TestUser, 0)

		// Test nil query without AllowEmpty (should return no records, blocked for safety)
		require.NoError(t, database.Database[*TestUser](nil).WithQuery(nil).List(&users))
		require.Equal(t, 0, len(users), "nil query should be blocked by default")

		// Test empty struct without AllowEmpty (should return no records, blocked for safety)
		require.NoError(t, database.Database[*TestUser](nil).WithQuery(&TestUser{}).List(&users))
		require.Equal(t, 0, len(users), "empty struct should be blocked by default")

		// Test query with all empty string fields without AllowEmpty (should return no records)
		// This tests the second check point where all field values are empty strings
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "", Email: ""}).
			List(&users))
		require.Equal(t, 0, len(users), "query with all empty string fields should be blocked by default")

		// Test nil query with AllowEmpty=true (should return all records)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{AllowEmpty: true}).
			List(&users))
		require.Equal(t, 3, len(users))
		var foundU1, foundU2, foundU3 bool
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Age, u.Age)
				require.Equal(t, u1.Email, u.Email)
				require.Equal(t, u1.IsActive, u.IsActive)
			case u2.ID:
				foundU2 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Age, u.Age)
				require.Equal(t, u2.Email, u.Email)
				require.Equal(t, u2.IsActive, u.IsActive)
			case u3.ID:
				foundU3 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Age, u.Age)
				require.Equal(t, u3.Email, u.Email)
				require.Equal(t, u3.IsActive, u.IsActive)
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test empty struct with AllowEmpty=true (should return all records)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{}, types.QueryConfig{AllowEmpty: true}).
			List(&users))
		require.Equal(t, 3, len(users))
		foundU1, foundU2, foundU3 = false, false, false
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1 = true
			case u2.ID:
				foundU2 = true
			case u3.ID:
				foundU3 = true
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test query with all empty string fields with AllowEmpty=true (should return all records)
		// This tests the second check point with AllowEmpty=true
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "", Email: ""}, types.QueryConfig{AllowEmpty: true}).
			List(&users))
		require.Equal(t, 3, len(users), "query with all empty string fields should return all records when AllowEmpty=true")
		foundU1, foundU2, foundU3 = false, false, false
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1 = true
			case u2.ID:
				foundU2 = true
			case u3.ID:
				foundU3 = true
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test query with some empty and some non-empty fields (should work normally, not blocked)
		// Query: Name="user1" (non-empty), Email="" (empty)
		// Should return u1 (matches Name), not blocked because at least one field is non-empty
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Email: ""}).
			List(&users))
		require.Equal(t, 1, len(users), "query with some non-empty fields should work normally")
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)
		require.Equal(t, u1.Email, users[0].Email)

		// Test AllowEmpty with FuzzyMatch: should allow empty query when both are enabled
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{}, types.QueryConfig{
				AllowEmpty: true,
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "AllowEmpty should work with FuzzyMatch")

		// Test AllowEmpty with UseOr: should allow empty query when both are enabled
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{}, types.QueryConfig{
				AllowEmpty: true,
				UseOr:      true,
			}).
			List(&users))
		require.Equal(t, 3, len(users), "AllowEmpty should work with UseOr")

		// Test AllowEmpty=false explicitly (should be same as default)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{AllowEmpty: false}).
			List(&users))
		require.Equal(t, 0, len(users), "AllowEmpty=false should block empty queries")
	})

	t.Run("UseOr", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		users := make([]*TestUser, 0)

		// Test UseOr=false (default, AND logic): query with multiple fields should return records matching ALL conditions
		// u1: Name="user1", Age=18
		// u2: Name="user2", Age=19
		// u3: Name="user3", Age=20
		// Query: Name="user1" AND Age=19 should return 0 records (no user matches both)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Age: u2.Age}, types.QueryConfig{UseOr: false}).
			List(&users))
		require.Equal(t, 0, len(users))

		// Test UseOr=false (default, AND logic): query with multiple fields matching same record
		// Query: Name="user1" AND Age=18 should return 1 record (u1 matches both)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Age: u1.Age}, types.QueryConfig{UseOr: false}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)
		require.Equal(t, u1.Age, users[0].Age)

		// Test UseOr=true (OR logic): query with multiple fields should return records matching ANY condition
		// Query: Name="user1" OR Age=19 should return 2 records (u1 matches Name, u2 matches Age)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Age: u2.Age}, types.QueryConfig{UseOr: true}).
			List(&users))
		require.Equal(t, 2, len(users))
		var foundU1, foundU2 bool
		for _, u := range users {
			if u.ID == u1.ID {
				foundU1 = true
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Age, u.Age)
			}
			if u.ID == u2.ID {
				foundU2 = true
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Age, u.Age)
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")

		// Test UseOr=true with three fields: Name="user1" OR Email="user2@example.com" OR Age=20
		// Should return all 3 records (u1 matches Name, u2 matches Email, u3 matches Age)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name, Email: u2.Email, Age: u3.Age}, types.QueryConfig{UseOr: true}).
			List(&users))
		require.Equal(t, 3, len(users))
		var foundU1_2, foundU2_2, foundU3 bool
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1_2 = true
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Email, u.Email)
				require.Equal(t, u1.Age, u.Age)
			case u2.ID:
				foundU2_2 = true
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Email, u.Email)
				require.Equal(t, u2.Age, u.Age)
			case u3.ID:
				foundU3 = true
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Email, u.Email)
				require.Equal(t, u3.Age, u.Age)
			}
		}
		require.True(t, foundU1_2, "should find u1")
		require.True(t, foundU2_2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test UseOr=true with single field (should work same as UseOr=false for single field)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name}, types.QueryConfig{UseOr: true}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)

		// Test UseOr=true with FuzzyMatch: Name LIKE "%user%" OR Email LIKE "%example%"
		// Should return all 3 records (all match both patterns)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: "user", Email: "example"}, types.QueryConfig{
				UseOr:      true,
				FuzzyMatch: true,
			}).
			List(&users))
		require.Equal(t, 3, len(users))
		foundU1_2, foundU2_2, foundU3 = false, false, false
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1_2 = true
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Email, u.Email)
				require.Equal(t, u1.Age, u.Age)
			case u2.ID:
				foundU2_2 = true
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Email, u.Email)
				require.Equal(t, u2.Age, u.Age)
			case u3.ID:
				foundU3 = true
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Email, u.Email)
				require.Equal(t, u3.Age, u.Age)
			}
		}
		require.True(t, foundU1_2, "should find u1")
		require.True(t, foundU2_2, "should find u2")
		require.True(t, foundU3, "should find u3")
	})

	t.Run("RawQuery", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)
		users := make([]*TestUser, 0)

		// Test RawQuery with nil query: age > 18
		// Should return u2 (age=19) and u3 (age=20)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "age > ?",
				RawQueryArgs: []any{18},
			}).
			List(&users))
		require.Equal(t, 2, len(users))
		var foundU2, foundU3 bool
		for _, u := range users {
			if u.ID == u2.ID {
				foundU2 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Age, u.Age)
				require.Equal(t, u2.Email, u.Email)
				require.Equal(t, u2.IsActive, u.IsActive)
			}
			if u.ID == u3.ID {
				foundU3 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Age, u.Age)
				require.Equal(t, u3.Email, u.Email)
				require.Equal(t, u3.IsActive, u.IsActive)
			}
		}
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test RawQuery with empty struct query: age >= 19
		// Should return u2 (age=19) and u3 (age=20)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{}, types.QueryConfig{
				RawQuery:     "age >= ?",
				RawQueryArgs: []any{19},
			}).
			List(&users))
		require.Equal(t, 2, len(users))
		foundU2, foundU3 = false, false
		for _, u := range users {
			if u.ID == u2.ID {
				foundU2 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Age, u.Age)
				require.Equal(t, u2.Email, u.Email)
			}
			if u.ID == u3.ID {
				foundU3 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Age, u.Age)
				require.Equal(t, u3.Email, u.Email)
			}
		}
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test RawQuery with multiple conditions: age BETWEEN ? AND ?
		// Should return u2 (age=19)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "age BETWEEN ? AND ?",
				RawQueryArgs: []any{19, 19},
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.NotEmpty(t, users[0].ID)
		require.NotEmpty(t, users[0].CreatedAt)
		require.NotEmpty(t, users[0].UpdatedAt)
		require.Equal(t, u2.ID, users[0].ID)
		require.Equal(t, u2.Name, users[0].Name)
		require.Equal(t, u2.Age, users[0].Age)
		require.Equal(t, u2.Email, users[0].Email)
		require.Equal(t, u2.IsActive, users[0].IsActive)

		// Test RawQuery with string condition: name = ?
		// Should return u1 (name="user1")
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "name = ?",
				RawQueryArgs: []any{u1.Name},
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.NotEmpty(t, users[0].ID)
		require.NotEmpty(t, users[0].CreatedAt)
		require.NotEmpty(t, users[0].UpdatedAt)
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)
		require.Equal(t, u1.Age, users[0].Age)
		require.Equal(t, u1.Email, users[0].Email)
		require.Equal(t, u1.IsActive, users[0].IsActive)

		// Test RawQuery with OR condition: name = ? OR age = ?
		// Should return u1 (name="user1") and u2 (age=19)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "name = ? OR age = ?",
				RawQueryArgs: []any{u1.Name, u2.Age},
			}).
			List(&users))
		require.Equal(t, 2, len(users))
		foundU1, foundU2 := false, false
		for _, u := range users {
			if u.ID == u1.ID {
				foundU1 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Age, u.Age)
				require.Equal(t, u1.Email, u.Email)
				require.Equal(t, u1.IsActive, u.IsActive)
			}
			if u.ID == u2.ID {
				foundU2 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Age, u.Age)
				require.Equal(t, u2.Email, u.Email)
				require.Equal(t, u2.IsActive, u.IsActive)
			}
		}
		require.True(t, foundU1, "should find u1")
		require.True(t, foundU2, "should find u2")

		// Test RawQuery with IN clause: age IN (?)
		// Should return u1 (age=18) and u3 (age=20)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "age IN (?)",
				RawQueryArgs: []any{[]int{18, 20}},
			}).
			List(&users))
		require.Equal(t, 2, len(users))
		var foundU1_2, foundU3_2 bool
		for _, u := range users {
			if u.ID == u1.ID {
				foundU1_2 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u1.Name, u.Name)
				require.Equal(t, u1.Age, u.Age)
				require.Equal(t, u1.Email, u.Email)
				require.Equal(t, u1.IsActive, u.IsActive)
			}
			if u.ID == u3.ID {
				foundU3_2 = true
				require.NotEmpty(t, u.ID)
				require.NotEmpty(t, u.CreatedAt)
				require.NotEmpty(t, u.UpdatedAt)
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Age, u.Age)
				require.Equal(t, u3.Email, u.Email)
				require.Equal(t, u3.IsActive, u.IsActive)
			}
		}
		require.True(t, foundU1_2, "should find u1")
		require.True(t, foundU3_2, "should find u3")

		// Test RawQuery with AND condition: name = ? AND age = ?
		// Should return u1 (name="user1" AND age=18)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "name = ? AND age = ?",
				RawQueryArgs: []any{u1.Name, u1.Age},
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Name, users[0].Name)
		require.Equal(t, u1.Age, users[0].Age)

		// Test RawQuery with AND condition that matches no records: name = ? AND age = ?
		// Should return 0 records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "name = ? AND age = ?",
				RawQueryArgs: []any{u1.Name, u2.Age},
			}).
			List(&users))
		require.Equal(t, 0, len(users))

		// Test RawQuery with no matching condition: age > 100
		// Should return 0 records
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "age > ?",
				RawQueryArgs: []any{100},
			}).
			List(&users))
		require.Equal(t, 0, len(users))

		// Test RawQuery with empty RawQueryArgs (should work when query has no placeholders)
		// Query: age = 18 (hardcoded value, no placeholders)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "age = 18",
				RawQueryArgs: nil,
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Age, users[0].Age)

		// Test RawQuery with empty RawQueryArgs slice (should work when query has no placeholders)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "age = 18",
				RawQueryArgs: []any{},
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u1.ID, users[0].ID)
		require.Equal(t, u1.Age, users[0].Age)

		// Test RawQuery with non-nil query: RawQuery should ignore model fields
		// RawQuery: age > 18, Query: Name="user1"
		// Should return u2 and u3 (age > 18), ignoring Name condition
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name}, types.QueryConfig{
				RawQuery:     "age > ?",
				RawQueryArgs: []any{18},
			}).
			List(&users))
		require.Equal(t, 2, len(users), "RawQuery should ignore model fields when both are provided")
		foundU2, foundU3 = false, false
		for _, u := range users {
			if u.ID == u2.ID {
				foundU2 = true
				require.Equal(t, u2.Name, u.Name)
				require.Equal(t, u2.Age, u.Age)
				require.Equal(t, u2.Email, u.Email)
			}
			if u.ID == u3.ID {
				foundU3 = true
				require.Equal(t, u3.Name, u.Name)
				require.Equal(t, u3.Age, u.Age)
				require.Equal(t, u3.Email, u.Email)
			}
		}
		require.True(t, foundU2, "should find u2")
		require.True(t, foundU3, "should find u3")

		// Test RawQuery with complex condition: (name = ? OR email = ?) AND age >= ?
		// Should return u2 (email="user2@example.com" AND age=19)
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "(name = ? OR email = ?) AND age >= ?",
				RawQueryArgs: []any{u2.Name, u2.Email, 19},
			}).
			List(&users))
		require.Equal(t, 1, len(users))
		require.Equal(t, u2.ID, users[0].ID)
		require.Equal(t, u2.Name, users[0].Name)
		require.Equal(t, u2.Age, users[0].Age)
		require.Equal(t, u2.Email, users[0].Email)

		// Test RawQuery with LIKE pattern: name LIKE ?
		// Should return all 3 records (all names contain "user")
		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(nil, types.QueryConfig{
				RawQuery:     "name LIKE ?",
				RawQueryArgs: []any{"%user%"},
			}).
			List(&users))
		require.Equal(t, 3, len(users))
		var foundU1_3, foundU2_3, foundU3_3 bool
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				foundU1_3 = true
			case u2.ID:
				foundU2_3 = true
			case u3.ID:
				foundU3_3 = true
			}
		}
		require.True(t, foundU1_3, "should find u1")
		require.True(t, foundU2_3, "should find u2")
		require.True(t, foundU3_3, "should find u3")
	})
}

func TestDatabaseWithCursor(t *testing.T) {
	defer cleanupTestData()

	t.Run("NextPage", func(t *testing.T) {
		defer cleanupTestData()
		count := 100
		data := make([]*TestUser, 0, count)
		for i := range count {
			name := fmt.Sprintf("user%05d", i)
			data = append(data, &TestUser{Name: name, Base: model.Base{ID: name}})
		}
		require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1000).Create(data...))

		// Get first record as starting cursor
		u := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).First(u))
		cursorValue := u.ID
		require.Equal(t, "user00000", cursorValue, "first record should be user00000")

		// Test pagination: fetch next pages
		users := make([]*TestUser, 0)
		for i := range 10 {
			users = make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).
				WithLimit(1).
				WithCursor(cursorValue, true).
				List(&users))
			require.Equal(t, 1, len(users), "should return 1 record per page")
			expectedID := fmt.Sprintf("user%05d", i+1)
			require.Equal(t, expectedID, users[0].ID, "should fetch next record in ascending order")
			cursorValue = users[0].ID
		}
	})

	t.Run("PreviousPage", func(t *testing.T) {
		defer cleanupTestData()
		count := 100
		data := make([]*TestUser, 0, count)
		for i := range count {
			name := fmt.Sprintf("user%05d", i)
			data = append(data, &TestUser{Name: name, Base: model.Base{ID: name}})
		}
		require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1000).Create(data...))

		// Get last record as starting cursor
		u := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Last(u))
		cursorValue := u.ID
		require.Equal(t, fmt.Sprintf("user%05d", count-1), cursorValue, "last record should be user00099")

		// Test pagination: fetch previous pages
		users := make([]*TestUser, 0)
		for i := range 10 {
			users = make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).
				WithLimit(1).
				WithCursor(cursorValue, false).
				List(&users))
			require.Equal(t, 1, len(users), "should return 1 record per page")
			expectedID := fmt.Sprintf("user%05d", count-2-i)
			require.Equal(t, expectedID, users[0].ID, "should fetch previous record in descending order")
			cursorValue = users[0].ID
		}
	})

	t.Run("CustomField", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// Test cursor pagination with custom field (created_at)
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 3, len(users))

		// Get first record's created_at as cursor
		// Format time to match database format (YYYY-MM-DD HH:MM:SS.ffffff)
		firstUser := users[0]
		require.NotNil(t, firstUser.CreatedAt, "first user should have created_at")
		cursorValue := firstUser.CreatedAt.Format("2006-01-02 15:04:05.000000")

		// Fetch next page using created_at as cursor field
		nextUsers := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithLimit(1).
			WithCursor(cursorValue, true, "created_at").
			List(&nextUsers))
		if len(nextUsers) > 0 {
			require.NotEqual(t, firstUser.ID, nextUsers[0].ID, "should fetch different record when available")
			require.NotNil(t, nextUsers[0].CreatedAt, "next user should have created_at")
			require.True(t, nextUsers[0].CreatedAt.After(*firstUser.CreatedAt) ||
				nextUsers[0].CreatedAt.Equal(*firstUser.CreatedAt),
				"next record should have created_at >= cursor value")
		}
	})

	t.Run("EmptyCursor", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// Test with empty cursor value (should be ignored)
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithLimit(10).
			WithCursor("", true).
			List(&users))
		require.Equal(t, 3, len(users), "empty cursor should be ignored, return all records")
	})

	t.Run("Combined", func(t *testing.T) {
		defer cleanupTestData()
		setupTestData(t)

		// Test cursor pagination combined with WithQuery
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name}).
			List(&users))
		require.Equal(t, 1, len(users))

		cursorValue := users[0].ID
		nextUsers := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).
			WithQuery(&TestUser{Name: u1.Name}).
			WithLimit(1).
			WithCursor(cursorValue, true).
			List(&nextUsers))
		require.Equal(t, 0, len(nextUsers), "no more records after cursor with query condition")
	})

	t.Run("MultiplePages", func(t *testing.T) {
		defer cleanupTestData()
		count := 50
		data := make([]*TestUser, 0, count)
		for i := range count {
			name := fmt.Sprintf("user%05d", i)
			data = append(data, &TestUser{Name: name, Base: model.Base{ID: name}})
		}
		require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1000).Create(data...))

		// Test pagination with page size > 1
		pageSize := 10
		cursorValue := ""
		allFetched := make([]string, 0)

		for range 5 {
			users := make([]*TestUser, 0)
			db := database.Database[*TestUser](nil).WithLimit(pageSize)
			if cursorValue != "" {
				db = db.WithCursor(cursorValue, true)
			}
			require.NoError(t, db.List(&users))
			require.LessOrEqual(t, len(users), pageSize, "should not exceed page size")

			if len(users) == 0 {
				break
			}

			for _, u := range users {
				allFetched = append(allFetched, u.ID)
			}
			cursorValue = users[len(users)-1].ID
		}

		require.Greater(t, len(allFetched), 0, "should fetch at least some records")
		// Verify no duplicates
		seen := make(map[string]bool)
		for _, id := range allFetched {
			require.False(t, seen[id], "should not have duplicate records: %s", id)
			seen[id] = true
		}
	})
}

func TestDatabaseWithSelect(t *testing.T) {
	defer cleanupTestData()

	// No effect on "Create"
	t.Run("Create", func(t *testing.T) {
		t.Run("with existing column", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name").Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 3)
		})
		t.Run("with non-existing column", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("notexists").Create(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 3)
		})
	})

	// No effect on "Delete"
	t.Run("Delete", func(t *testing.T) {
		t.Run("with existing column", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name").Create(ul...))
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name").Delete(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 0)
		})
		t.Run("with non-existing column", func(t *testing.T) {
			defer cleanupTestData()
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("notexists").Create(ul...))
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("notexists").Delete(ul...))
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 0)
		})
	})

	// Effect "Update"
	t.Run("Update", func(t *testing.T) {
		t.Run("with existing column", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			u1.Name = "user1_modified"
			u2.Name = "user2_modified"
			u3.Name = "user3_modified"
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name").Update(ul...))

			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Equal(t, "user1_modified", u11.Name)
			require.Equal(t, "user2_modified", u22.Name)
			require.Equal(t, "user3_modified", u33.Name)
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u3.Age, u33.Age)
			require.Equal(t, u1.Email, u11.Email)
			require.Equal(t, u2.Email, u22.Email)
			require.Equal(t, u3.Email, u33.Email)
			require.Equal(t, u1.IsActive, u11.IsActive)
			require.Equal(t, u2.IsActive, u22.IsActive)
			require.Equal(t, u3.IsActive, u33.IsActive)
		})
		t.Run("with different column", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			u1OldName, u2OldName, u3OldName := u1.Name, u2.Name, u3.Name
			u1.Name = "user1_modified"
			u2.Name = "user2_modified"
			u3.Name = "user3_modified"
			// Only update column "age", the modified name will not be updated.
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("age").Update(ul...))

			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Equal(t, u1OldName, u11.Name, "name should not be updated")
			require.Equal(t, u2OldName, u22.Name, "name should not be updated")
			require.Equal(t, u3OldName, u33.Name, "name should not be updated")
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u3.Age, u33.Age)
			require.Equal(t, u1.Email, u11.Email)
			require.Equal(t, u2.Email, u22.Email)
			require.Equal(t, u3.Email, u33.Email)
			require.Equal(t, u1.IsActive, u11.IsActive)
			require.Equal(t, u2.IsActive, u22.IsActive)
			require.Equal(t, u3.IsActive, u33.IsActive)
		})
		t.Run("with non-existing column", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			u1OldName, u2OldName, u3OldName := u1.Name, u2.Name, u3.Name
			u1.Name = "user1_modified"
			u2.Name = "user2_modified"
			u3.Name = "user3_modified"
			// The non-existing fields will be ignored, and only default columns will be selected.
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("nonexistent").Update(ul...))

			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Equal(t, u1OldName, u11.Name, "name should not be updated")
			require.Equal(t, u2OldName, u22.Name, "name should not be updated")
			require.Equal(t, u3OldName, u33.Name, "name should not be updated")
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u3.Age, u33.Age)
			require.Equal(t, u1.Email, u11.Email)
			require.Equal(t, u2.Email, u22.Email)
			require.Equal(t, u3.Email, u33.Email)
			require.Equal(t, u1.IsActive, u11.IsActive)
			require.Equal(t, u2.IsActive, u22.IsActive)
			require.Equal(t, u3.IsActive, u33.IsActive)
		})
		t.Run("with multiple columns", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			u1.Name = "user1_modified"
			u1.Age = 25
			u2.Name = "user2_modified"
			u2.Age = 26
			u3.Name = "user3_modified"
			u3.Age = 27
			// Update both "name" and "age" columns.
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name", "age").Update(ul...))

			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Equal(t, "user1_modified", u11.Name)
			require.Equal(t, "user2_modified", u22.Name)
			require.Equal(t, "user3_modified", u33.Name)
			require.Equal(t, 25, u11.Age)
			require.Equal(t, 26, u22.Age)
			require.Equal(t, 27, u33.Age)
			require.Equal(t, u1.Email, u11.Email, "email should not be updated")
			require.Equal(t, u2.Email, u22.Email, "email should not be updated")
			require.Equal(t, u3.Email, u33.Email, "email should not be updated")
		})
	})

	// Effect "List"
	t.Run("List", func(t *testing.T) {
		t.Run("with single column", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			// Only select column "name", other columns will be ignored.
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name").List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Equal(t, u1.Name, u11.Name)
			require.Equal(t, u2.Name, u22.Name)
			require.Equal(t, u3.Name, u33.Name)
			// Only select "name", fields "age" and "email" should be empty.
			require.Empty(t, u11.Age)
			require.Empty(t, u22.Age)
			require.Empty(t, u33.Age)
			require.Empty(t, u11.Email)
			require.Empty(t, u22.Email)
			require.Empty(t, u33.Email)
		})
		t.Run("with different column", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			// Only select column "age", other columns will be ignored.
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("age").List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Empty(t, u11.Name)
			require.Empty(t, u22.Name)
			require.Empty(t, u33.Name)
			// Only select "age", fields "name" and "email" should be empty.
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u3.Age, u33.Age)
			require.Empty(t, u11.Email)
			require.Empty(t, u22.Email)
			require.Empty(t, u33.Email)
		})
		t.Run("with multiple columns", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			// Select both "name" and "age" columns.
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).WithSelect("name", "age").List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			require.Equal(t, u1.Name, u11.Name)
			require.Equal(t, u2.Name, u22.Name)
			require.Equal(t, u3.Name, u33.Name)
			require.Equal(t, u1.Age, u11.Age)
			require.Equal(t, u2.Age, u22.Age)
			require.Equal(t, u3.Age, u33.Age)
			// Field "email" should be empty.
			require.Empty(t, u11.Email)
			require.Empty(t, u22.Email)
			require.Empty(t, u33.Email)
		})
		t.Run("with non-existing column", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			// Selecting non-existing column will cause error.
			users := make([]*TestUser, 0)
			require.Error(t, database.Database[*TestUser](nil).WithSelect("notexists").List(&users))
		})
		t.Run("with empty columns", func(t *testing.T) {
			defer cleanupTestData()
			setupTestData(t)
			// WithSelect with no columns should select only default columns.
			users := make([]*TestUser, 0)
			require.NoError(t, database.Database[*TestUser](nil).WithSelect().List(&users))
			require.Len(t, users, 3)
			u11, u22, u33 := findUsersByID(users)
			require.NotNil(t, u11)
			require.NotNil(t, u22)
			require.NotNil(t, u33)
			// Default columns (id, created_at, updated_at, etc.) should be present.
			require.NotEmpty(t, u11.ID)
			require.NotEmpty(t, u22.ID)
			require.NotEmpty(t, u33.ID)
			require.NotEmpty(t, u11.CreatedAt)
			require.NotEmpty(t, u22.CreatedAt)
			require.NotEmpty(t, u33.CreatedAt)
		})
	})
}
