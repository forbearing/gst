package dbmigrate_test

import (
	"testing"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/pkg/dbmigrate"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		dumper, err := dbmigrate.NewSchemaDumper()
		require.NoError(t, err)
		schema, err := dumper.Dump(config.DBMySQL, User{}, Group{})
		require.NoError(t, err)

		err = dbmigrate.Migrate([]string{schema}, config.DBMySQL,
			&dbmigrate.DatabaseConfig{
				Host:     "127.0.0.1",
				Port:     3307,
				Username: "test",
				Password: "test",
				Database: "test",
			},
			&dbmigrate.MigrateOption{
				DryRun: true,
			})
		require.NoError(t, err)
	})

	t.Run("postgres", func(t *testing.T) {
		dumper, err := dbmigrate.NewSchemaDumper()
		require.NoError(t, err)
		schema, err := dumper.Dump(config.DBPostgres, User{}, Group{})
		require.NoError(t, err)

		err = dbmigrate.Migrate([]string{schema}, config.DBPostgres,
			&dbmigrate.DatabaseConfig{
				Host:     "127.0.0.1",
				Port:     5432,
				Username: "test",
				Password: "test",
				Database: "test",
				SSLMode:  "disable",
			},
			&dbmigrate.MigrateOption{
				DryRun: true,
			},
		)
		require.NoError(t, err)
	})

	t.Run("sqlite", func(t *testing.T) {
		dumper, err := dbmigrate.NewSchemaDumper()
		require.NoError(t, err)
		schema, err := dumper.Dump(config.DBSqlite, User{}, Group{})
		require.NoError(t, err)

		err = dbmigrate.Migrate([]string{schema}, config.DBSqlite,
			&dbmigrate.DatabaseConfig{
				Database: "test",
			},
			&dbmigrate.MigrateOption{
				DryRun: true,
			})
		require.NoError(t, err)
	})
}
