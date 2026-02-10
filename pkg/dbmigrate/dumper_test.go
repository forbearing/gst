package dbmigrate_test

import (
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
