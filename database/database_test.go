package database_test

import (
	"errors"
	"os"
	"testing"

	"github.com/forbearing/gst/bootstrap"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/database/sqlite"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
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

	// os.Setenv(config.DATABASE_TYPE, string(config.DBMySQL))
	// os.Setenv(config.MYSQL_DATABASE, "test")
	// os.Setenv(config.MYSQL_USERNAME, "test")
	// os.Setenv(config.MYSQL_PASSWORD, "test")

	model.Register[*TestUser]()
	model.Register[*TestProduct]()
	model.Register[*TestCategory]()

	if err := bootstrap.Bootstrap(); err != nil {
		panic(err)
	}
}

func TestDatabaseOperation(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(u1))
		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(1), *count)

		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count)

		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(0), *count)

		// check the create hook that will effect original data.
		require.Equal(t, remarkUserCreateBefore, *u1.Remark)
		require.Equal(t, remarkUserCreateBefore, *u2.Remark)
		require.Equal(t, remarkUserCreateBefore, *u3.Remark)

		// check the created data is same as the original data.
		u1.Remark, u2.Remark, u3.Remark = nil, nil, nil
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))

		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
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
		// check the created data is not empty
		require.NotNil(t, u11)
		require.NotNil(t, u22)
		require.NotNil(t, u33)
		require.NotEmpty(t, u11.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u22.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u33.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u1.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u2.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u3.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u11.ID)        // id should not be empty
		require.NotEmpty(t, u22.ID)        // id should not be empty
		require.NotEmpty(t, u33.ID)        // id should not be empty

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

		// Check create empty resources
		require.NoError(t, database.Database[*TestUser](nil).Create(nil))
		require.NoError(t, database.Database[*TestUser](nil).Create([]*TestUser{nil, nil, nil}...))
		require.NoError(t, database.Database[*TestUser](nil).Create([]*TestUser{nil, u1, nil}...))
	})

	t.Run("Delete", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()

		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count)

		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(0), *count)

		// Check delete empty resources
		require.NoError(t, database.Database[*TestUser](nil).Delete(nil))
		require.NoError(t, database.Database[*TestUser](nil).Delete([]*TestUser{nil, nil, nil}...))
		require.NoError(t, database.Database[*TestUser](nil).Delete([]*TestUser{nil, u1, nil}...))
	})

	t.Run("Update", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Update(ul...))
		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count)

		// check the update hook result.
		require.Equal(t, remarkUserUpdateBefore, *u1.Remark)
		require.Equal(t, remarkUserUpdateBefore, *u2.Remark)
		require.Equal(t, remarkUserUpdateBefore, *u3.Remark)

		// check the data upset in the database.
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
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
		require.NotEmpty(t, u11.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u22.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u33.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u1.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u2.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u3.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u11.ID)        // id should not be empty
		require.NotEmpty(t, u22.ID)        // id should not be empty
		require.NotEmpty(t, u33.ID)        // id should not be empty
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

		// check the data upset in the database again.
		require.NoError(t, database.Database[*TestUser](nil).Update(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count)

		users = make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
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
		require.NotEmpty(t, u11.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u22.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u33.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u1.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u2.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u3.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u11.ID)        // id should not be empty
		require.NotEmpty(t, u22.ID)        // id should not be empty
		require.NotEmpty(t, u33.ID)        // id should not be empty
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

		// Check update empty resources.
		require.NoError(t, database.Database[*TestUser](nil).Update(nil))
		require.NoError(t, database.Database[*TestUser](nil).Update([]*TestUser{nil, nil, nil}...))
		require.NoError(t, database.Database[*TestUser](nil).Update([]*TestUser{nil, u1, nil}...))
	})

	t.Run("UpdateByID", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(u1))

		// Check the data created in the database.
		u := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
		require.NotNil(t, u)
		require.NotEmpty(t, u.CreatedAt)
		require.NotEmpty(t, u.UpdatedAt)
		require.NotEmpty(t, u.ID)
		require.Equal(t, u1.Name, u.Name)
		require.Equal(t, u1.Age, u.Age)
		require.Equal(t, u1.Email, u.Email)

		newName := "user1_modified"
		require.NoError(t, database.Database[*TestUser](nil).UpdateByID(u.ID, "name", newName))
		u = new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
		require.NotNil(t, u)
		require.NotEmpty(t, u.CreatedAt)
		require.NotEmpty(t, u.UpdatedAt)
		require.NotEmpty(t, u.ID)
		require.Equal(t, newName, u.Name)
		require.Equal(t, u1.Age, u.Age)
		require.Equal(t, u1.Email, u.Email)

		// Check empty id, name and value.
		require.NoError(t, database.Database[*TestUser](nil).UpdateByID("", "", nil))
		require.NoError(t, database.Database[*TestUser](nil).UpdateByID("", "name", nil))
		require.NoError(t, database.Database[*TestUser](nil).UpdateByID("id", "", nil))
	})

	t.Run("List", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))

		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 3, len(users))

		var u11, u22, u33 *TestUser
		for _, u := range users {
			switch u.ID {
			case u1.ID:
				u11 = u
			case u2.ID:
				u22 = u2
			case u3.ID:
				u33 = u3
			}
		}
		require.NotNil(t, u11)
		require.NotNil(t, u22)
		require.NotNil(t, u33)
		require.NotEmpty(t, u11.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u22.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u33.CreatedAt) // created_at should not be empty
		require.NotEmpty(t, u1.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u2.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u3.UpdatedAt)  // updated_at should not be empty
		require.NotEmpty(t, u11.ID)        // id should not be empty
		require.NotEmpty(t, u22.ID)        // id should not be empty
		require.NotEmpty(t, u33.ID)        // id should not be empty
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

		// the "users" is not empty, its contains 3 objects.
		require.Equal(t, 3, len(users))
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		// the "users" will be overwritten by "List".
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 0, len(users))
	})

	t.Run("Get", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		u := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Get(u, u1.ID))
		require.NotNil(t, u)
		require.NotEmpty(t, u.CreatedAt)
		require.NotEmpty(t, u.UpdatedAt)
		require.NotEmpty(t, u.ID)
		require.Equal(t, u1.Name, u.Name)
		require.Equal(t, u1.Age, u.Age)
		require.Equal(t, u1.Email, u.Email)
	})

	t.Run("First", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		u := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).First(u))
		require.NotNil(t, u)
		require.NotEmpty(t, u.CreatedAt)
		require.NotEmpty(t, u.UpdatedAt)
		require.NotEmpty(t, u.ID)
		require.Equal(t, u1.Name, u.Name)
		require.Equal(t, u1.Age, u.Age)
		require.Equal(t, u1.Email, u.Email)
	})

	t.Run("Last", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		u := new(TestUser)
		require.NoError(t, database.Database[*TestUser](nil).Last(u))
		require.NotNil(t, u)
		require.NotEmpty(t, u.CreatedAt)
		require.NotEmpty(t, u.UpdatedAt)
		require.NotEmpty(t, u.ID)
		require.Equal(t, u3.Name, u.Name)
		require.Equal(t, u3.Age, u.Age)
		require.Equal(t, u3.Email, u.Email)
	})

	t.Run("Count", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))

		count := new(int64)
		require.NoError(t, database.Database[*TestUser](nil).Count(count))
		require.Equal(t, int64(3), *count)
	})

	t.Run("Cleanup", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Create(ul...))
		require.NoError(t, database.Database[*TestUser](nil).Cleanup())
	})

	t.Run("Health", func(t *testing.T) {
		require.NoError(t, database.Database[*TestUser](nil).Health())
	})
}

func TestDatabaseOptions(t *testing.T) {
	t.Run("WithDB", func(t *testing.T) {
		path2 := "/tmp/test2.db"
		path3 := "/tmp/test3.db"
		defer func() {
			_ = os.Remove(path2)
			_ = os.Remove(path3)
		}()

		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

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

		// create resources in `db2`.
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Create(ul...))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
		require.Equal(t, 3, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
		require.Equal(t, 0, len(users))

		// create resources in `db3`.
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).Create(ul...))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
		require.Equal(t, 3, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
		require.Equal(t, 3, len(users))

		// delete resources in default db
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 0, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
		require.Equal(t, 3, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
		require.Equal(t, 3, len(users))

		// delete resources in db2
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 0, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
		require.Equal(t, 0, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
		require.Equal(t, 3, len(users))

		// delete resources in db3
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).Delete(ul...))
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 0, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).List(&users))
		require.Equal(t, 0, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).List(&users))
		require.Equal(t, 0, len(users))
	})

	t.Run("WithTable", func(t *testing.T) {
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

		// manually migrate the database.
		require.NoError(t, db2.AutoMigrate(TestUser{}))
		require.NoError(t, db3.AutoMigrate(TestUser{}))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").Create(ul...))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).WithTable("test_users").Create(ul...))

		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db2).WithTable("test_users").List(&users))
		require.Equal(t, 3, len(users))
		require.NoError(t, database.Database[*TestUser](nil).WithDB(db3).WithTable("test_users").List(&users))
		require.Equal(t, 3, len(users))
	})

	t.Run("WithTx", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

		// transaction success, created three users.
		err := database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
			require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
			return nil
		})
		_ = err
		require.NoError(t, err)
		users := make([]*TestUser, 0)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 3, len(users))

		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))
		err = database.Database[*TestUser](nil).TransactionFunc(func(tx any) error {
			require.NoError(t, database.Database[*TestUser](nil).WithTx(tx).Create(ul...))
			return errors.New("custom error")
		})
		require.Error(t, err)
		require.NoError(t, database.Database[*TestUser](nil).List(&users))
		require.Equal(t, 0, len(users))

		// transaction failed, not created.
	})

	t.Run("WithBatchSize", func(t *testing.T) {
		defer func() {
			_ = database.Database[*TestUser](nil).Delete(ul...)
		}()
		require.NoError(t, database.Database[*TestUser](nil).Delete(ul...))

		require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1000).Create(ul...))
		require.NoError(t, database.Database[*TestUser](nil).WithBatchSize(1000).Delete(ul...))
	})
}
