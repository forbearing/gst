package database

import (
	"github.com/forbearing/gst/types"
	"gorm.io/gorm"
)

// BuildSQL builds SQL for the operations run inside fn without executing database I/O.
// It preserves placeholders in SQL and returns bound values separately in SQLStatement.Vars.
// BuildSQL is intended for CRUD, read, cleanup, and health-check SQL generation; transaction
// helpers are not supported because they manage real database transaction control flow.
func (db *database[M]) BuildSQL(fn func(db types.Database[M]) error) (statements []types.SQLStatement, err error) {
	if fn == nil {
		return nil, ErrNilSQLBuilder
	}

	db.mu.Lock()
	db.dryRun = true
	db.buildingSQL = true
	db.sqlStatements = &statements
	db.mu.Unlock()

	defer db.reset()

	if err = fn(db); err != nil {
		return statements, err
	}
	return statements, nil
}

func (db *database[M]) collectSQL(tx *gorm.DB) error {
	if tx == nil {
		return nil
	}
	if !db.buildingSQL {
		return tx.Error
	}
	if tx.Statement != nil {
		if sql := tx.Statement.SQL.String(); len(sql) > 0 {
			vars := append([]any(nil), tx.Statement.Vars...)
			db.mu.Lock()
			if db.sqlStatements != nil {
				*db.sqlStatements = append(*db.sqlStatements, types.SQLStatement{
					SQL:  sql,
					Vars: vars,
				})
			}
			db.mu.Unlock()
		}
	}
	return tx.Error
}
