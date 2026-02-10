package dbmigrate

import (
	"database/sql"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/forbearing/gst/config"
	"github.com/maxrichie5/go-sqlfmt/sqlfmt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SchemaDumper struct {
	log  logger.Interface
	db   *sql.DB
	mock sqlmock.Sqlmock

	mu sync.Mutex
}

func NewSchemaDumper() (*SchemaDumper, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	return &SchemaDumper{
		db:   db,
		mock: mock,
		log:  &dumperLogger{},
	}, nil
}

func (s *SchemaDumper) Dump(driver config.DBType, dst ...any) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var dialector gorm.Dialector
	var tableOptions string

	switch driver {
	case config.DBMySQL:
		dialector = mysql.New(mysql.Config{Conn: s.db, SkipInitializeWithVersion: true})
		tableOptions = "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin"
	case config.DBPostgres:
		dialector = postgres.New(postgres.Config{Conn: s.db, PreferSimpleProtocol: true})
	case config.DBSqlite:
		dialector = sqlite.New(sqlite.Config{Conn: s.db})
		// GORM sqlite driver might ping to check version
		s.mock.ExpectQuery("select sqlite_version()").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("3.35.0"))
	}

	db, err := gorm.Open(dialector, &gorm.Config{DryRun: true, Logger: s.log})
	if err != nil {
		return "", err
	}
	if err = db.Set("gorm:table_options", tableOptions).Migrator().CreateTable(dst...); err != nil {
		return "", err
	}

	l, ok := s.log.(*dumperLogger)
	if !ok {
		return "", errors.New("invalid logger type")
	}
	sqls := l.SQLs
	if len(sqls) == 0 {
		return "", nil
	}

	var sb strings.Builder

	for _, sql := range sqls {
		if _, err := sb.WriteString(sqlfmt.Format(sql) + ";\n"); err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}

func (s *SchemaDumper) Close() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		err = s.db.Close()
		s.db = nil
		return err
	}

	return nil
}

//	func (s *SchemaDumper) Logger() logger.Interface {
//		s.mu.Lock()
//		defer s.mu.Unlock()
//
//		if s.log != nil {
//			return s.log
//		}
//
//		s.log = &dumperLogger{}
//		return s.log
//	}
//
//	func (s *SchemaDumper) Conn() (*sql.DB, error) {
//		s.mu.Lock()
//		defer s.mu.Unlock()
//
//		if s.db != nil {
//			return s.db, nil
//		}
//
//		var err error
//		if s.db, s.mock, err = sqlmock.New(); err != nil {
//			return nil, err
//		}
//
//		return s.db, nil
//	}
