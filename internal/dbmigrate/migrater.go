package dbmigrate

import (
	"strings"

	"github.com/forbearing/gst/config"
	"github.com/sqldef/sqldef/v3"
	"github.com/sqldef/sqldef/v3/database"
	"github.com/sqldef/sqldef/v3/database/mysql"
	"github.com/sqldef/sqldef/v3/database/postgres"
	"github.com/sqldef/sqldef/v3/database/sqlite3"
	"github.com/sqldef/sqldef/v3/parser"
	"github.com/sqldef/sqldef/v3/schema"
)

type DatabaseConfig struct {
	Database string
	Username string
	Password string
	Host     string
	Port     int
}

type MigrateOption struct {
	Schemas []string
	DryRun  bool
	Export  bool
}

func Migrate(schemas []string, dbtyp config.DBType, cfg *DatabaseConfig, opt *MigrateOption) (err error) {
	if len(schemas) == 0 {
		return nil
	}
	if cfg == nil {
		return nil
	}
	if opt == nil {
		opt = &MigrateOption{}
	}

	dbcfg := database.Config{
		DbName:   cfg.Database,
		User:     cfg.Username,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
	}
	migOpt := &sqldef.Options{
		DryRun:      opt.DryRun,
		Export:      opt.Export,
		DesiredDDLs: strings.Join(schemas, ";\n"),
	}

	var db database.Database
	var parseMode parser.ParserMode
	var genMode schema.GeneratorMode

	switch dbtyp {
	case config.DBMySQL:
		db, err = mysql.NewDatabase(dbcfg)
		parseMode = parser.ParserModeMysql
		genMode = schema.GeneratorModeMysql
	case config.DBPostgres:
		db, err = postgres.NewDatabase(dbcfg)
		parseMode = parser.ParserModePostgres
		genMode = schema.GeneratorModePostgres
	case config.DBSqlite:
		db, err = sqlite3.NewDatabase(dbcfg)
		parseMode = parser.ParserModeSQLite3
		genMode = schema.GeneratorModeSQLite3
	}
	if err != nil {
		return err
	}
	defer db.Close()

	sqlParser := database.NewParser(parseMode)
	sqldef.Run(genMode, db, sqlParser, migOpt)

	return nil
}
