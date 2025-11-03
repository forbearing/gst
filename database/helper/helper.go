package helper

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/util"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// startedTable is an atomic flag to ensure table processing goroutine starts only once
var startedTable int32

// initedTable is a concurrent map that tracks initialized tables by their unique key (table_name:db_name)
// It is used by the record processing goroutine to wait for table creation before inserting records
var initedTable = cmap.New[string]()

// InitDatabase initializes database tables and records with asynchronous processing support.
// It creates tables and inserts records that are registered via Register() or RegisterTo() functions.
// The function supports concurrent model registration at any stage - before, during, or after InitDatabase execution.
//
// Key features:
//   - Asynchronous table creation and record insertion using goroutines and channels
//   - Thread-safe concurrent model registration support
//   - Automatic handling of both default database and custom database instances
//   - Real-time processing of models registered during initialization
//
// NOTE: The version of gorm.io/driver/postgres lower than v1.5.4 have some issues.
// More details see: https://github.com/go-gorm/gorm/issues/6886
func InitDatabase(db *gorm.DB, dbmap map[string]*gorm.DB) (err error) {
	// Install GORM OpenTelemetry tracing plugin
	if err = db.Use(otelgorm.NewPlugin()); err != nil {
		zap.S().Warnw("failed to install GORM OpenTelemetry tracing plugin", "error", err)
	}

	// Install tracing plugin for custom databases
	for _, customDB := range dbmap {
		if err = customDB.Use(otelgorm.NewPlugin()); err != nil {
			zap.S().Warnw("failed to install GORM OpenTelemetry tracing plugin for custom DB", "error", err)
		}
	}

	if atomic.CompareAndSwapInt32(&startedTable, 0, 1) {
		go func() {
			for {
				select {
				case m := <-model.TableChan:
					// create table automatically in default database.
					begin := time.Now()
					typ := reflect.TypeOf(m).Elem()
					if err = db.Table(m.GetTableName()).AutoMigrate(m); err != nil {
						err = errors.Wrap(err, fmt.Sprintf("failed to create table(%s)", typ.String()))
						panic(err)
					}
					zap.S().Infow("database create table", "cost", util.FormatDurationSmart(time.Since(begin)))

					initedTable.Set(typ.String(), "")

				case v := <-model.TableDBChan:
					// create table automatically with custom database.
					begin := time.Now()
					handler := db
					if val, exists := dbmap[strings.ToLower(v.DBName)]; exists {
						handler = val
					}
					m := v.Table
					typ := reflect.TypeOf(m).Elem()
					if err = handler.Table(m.GetTableName()).AutoMigrate(m); err != nil {
						err = errors.Wrap(err, fmt.Sprintf("failed to create table(%s)", typ.String()))
						panic(err)
					}
					zap.S().Infow("database create table", "cost", util.FormatDurationSmart(time.Since(begin)))

					initedTable.Set(typ.String(), v.DBName)

				case r := <-model.RecordChan:
					// create the table records that must be pre-exists before database curds.
					// NOTE: we should always creates records after table migration finished.
					typ := reflect.TypeOf(r.Table).Elem()
					for {
						dbname, e := initedTable.Get(typ.String())
						if e && dbname == r.DBName {
							break
						}
						time.Sleep(300 * time.Millisecond)
					}

					begin := time.Now()
					handler := db
					if val, exists := dbmap[strings.ToLower(r.DBName)]; exists {
						handler = val
					}
					if err = handler.Table(r.Table.GetTableName()).Save(r.Rows).Error; err != nil {
						err = errors.Wrap(err, "failed to create table records")
						panic(err)
					}
					zap.S().Infow("database create table records", "cost", util.FormatDurationSmart(time.Since(begin)))

				}
			}
		}()
	}

	// set default database to 'Default'.
	database.DB = db

	return nil
}

// Transaction start a transaction as a block, return error will rollback, otherwise to commit.
// Transaction executes an arbitrary number of commands in fc within a transaction.
// On success the changes are committed; if an error occurs they are rolled back.
func Transaction(db *gorm.DB, fn func(tx *gorm.DB) error) error { return db.Transaction(fn) }

// Exec executes raw sql without return rows
func Exec(db *gorm.DB, sql string, values any) error { return db.Exec(sql, values).Error }
