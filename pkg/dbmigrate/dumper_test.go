package dbmigrate_test

import (
	"strings"
	"testing"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/pkg/dbmigrate"
	"github.com/stretchr/testify/require"
)

type User struct {
	Username string `json:"username"`
	Addr     string `json:"addr"`

	model.Base
}

type Group struct {
	Name string `json:"name"`

	model.Base
}

func TestDumper(t *testing.T) {
	dumper, err := dbmigrate.NewSchemaDumper()
	require.NoError(t, err)
	defer dumper.Close()

	t.Run("mysql", func(t *testing.T) {
		schema, err := dumper.Dump(config.DBMySQL, User{}, &Group{})
		require.NoError(t, err)
		require.NotEmpty(t, schema)
		// fmt.Println(schema)
	})

	t.Run("postgres", func(t *testing.T) {
		schema, err := dumper.Dump(config.DBPostgres, User{}, &Group{})
		require.NoError(t, err)
		require.NotEmpty(t, schema)
		// fmt.Println(schema)
		t.Log(schema)
	})

	t.Run("sqlite", func(t *testing.T) {
		schema, err := dumper.Dump(config.DBSqlite, User{}, &Group{})
		require.NoError(t, err)
		require.NotEmpty(t, schema)
		// fmt.Println(schema)
	})
}

func TestDumpOrder(t *testing.T) {
	dumper, err := dbmigrate.NewSchemaDumper()
	require.NoError(t, err)
	defer dumper.Close()

	// reflect.TypeOf(User{}).String() is "dbmigrate_test.User"
	// reflect.TypeOf(&Group{}).String() is "*dbmigrate_test.Group"
	// '*' (42) < 'd' (100), so &Group{} should be first.
	// However, if we use pointers for both:
	// *dbmigrate_test.Group vs *dbmigrate_test.User
	// G < U, so Group first.

	t.Run("stable_order", func(t *testing.T) {
		// Pass User then Group (as pointers)
		// Expected order: Group (sorted first), then User
		schema, err := dumper.Dump(config.DBMySQL, &User{}, &Group{})
		require.NoError(t, err)

		idxGroup := strings.Index(schema, "CREATE TABLE `groups`")
		idxUser := strings.Index(schema, "CREATE TABLE `users`")

		require.True(t, idxGroup != -1)
		require.True(t, idxUser != -1)
		require.True(t, idxGroup < idxUser, "Group should appear before User because *...Group < *...User")
	})
}
