package database

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/cache"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/logger"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	"github.com/stoewer/go-strcase"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	glogger "gorm.io/gorm/logger"
	"gorm.io/hints"
)

// references:

var (
	ErrInvalidDB           = errors.New("invalid database, maybe not initialized")
	ErrUnknowDBType        = errors.New("unknow database type, only support mysql or sqlite")
	ErrNotPtrStruct        = errors.New("model is not pointer to structure")
	ErrNotPtrSlice         = errors.New("not pointer to slice")
	ErrNotPtrInt64         = errors.New("not pointer to int64")
	ErrNotAddressableModel = errors.New("model is not addressable")
	ErrNotAddressableSlice = errors.New("slice is not addressable")
	ErrNotSetSlice         = errors.New("slice cannot set")
	ErrIDRequired          = errors.New("id is required")
	ErrManualRollback      = errors.New("manual rollback requested")
)

// migratedModelMap records the model already migrated to
// avoid duplicate migration and improve performance.
// Key is "dbIdentifier:modelType", value is "struct{}{}".
// dbIdentifier is the unique identifier of the database instance (e.g., pointer address of the underlying database connection).
var migratedModelMap sync.Map

var (
	DB *gorm.DB

	defaultLimit           = -1
	defaultBatchSize       = 1000
	defaultDeleteBatchSize = 10000
	defaultsColumns        = []string{
		"id",
		"created_by",
		"updated_by",
		"created_at",
		"updated_at",
		"deleted_at",
	}
)

// database inplement types.Database[T types.Model] interface.
type database[M types.Model] struct {
	ins *gorm.DB
	m   M
	typ reflect.Type
	ctx *types.DatabaseContext
	mu  sync.Mutex

	// options
	enablePurge *bool  // delete resource permanently, not only update deleted_at field, only works on 'Delete' method.
	enableCache bool   // using cache or not, only works 'List', 'Get', 'Count' method.
	tableName   string // support multiple custom table name, always used with the `WithDB` method.
	batchSize   int    // batch size for bulk operations. affects Create, Update, Delete.
	noHook      bool   // disable model hook.
	dryRun      bool   // dry run mode, preview SQL without executing

	// cursor pagination
	cursorField  string // field used for cursor pagination, default is "id"
	cursorValue  string // cursor value for pagination
	cursorNext   bool   // direction of cursor pagination, true for next page, false for previous page
	enableCursor bool   // enable cursor pagination

	// rollback control
	rollbackFunc func() error // rollback function for manual transaction control

	shouldAutoMigrate *bool
}

// reset resets the database instance to its initial state by clearing all query conditions,
// restoring default settings, and preparing for the next operation.
// This method must be called in all functions except option functions prefixed with 'With*'.
func (db *database[M]) reset() {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.enablePurge = nil
	db.enableCache = false
	db.tableName = ""
	db.batchSize = 0
	db.noHook = false
	db.shouldAutoMigrate = nil
	db.dryRun = false

	// reset cursor pagination fields
	db.cursorField = ""
	db.cursorValue = ""
	db.cursorNext = true
	db.enableCursor = false

	// reset rollback function
	db.rollbackFunc = nil
}

// prepare prepares the database instance for query execution by applying all configured
// query conditions, joins, and other settings to the underlying GORM database instance.
func (db *database[M]) prepare() error {
	if db.ins == nil || db.ins == new(gorm.DB) {
		return ErrInvalidDB
	}
	if db.shouldAutoMigrate != nil && *db.shouldAutoMigrate {
		if err := db.ins.AutoMigrate(new(M)); err != nil {
			return err
		}
	}

	db.typ = reflect.TypeOf(*new(M)).Elem()
	db.m = reflect.New(db.typ).Interface().(M) //nolint:errcheck
	// Set enablePurge based on model's Purge() method if not explicitly set by WithPurge().
	// Priority: WithPurge() > model.Purge() > default (soft delete)
	// - If WithPurge() was called, use the explicitly set value (highest priority)
	// - Otherwise, use model.Purge() to determine the default delete behavior
	// - model.Purge() returns true: hard delete (permanent deletion)
	// - model.Purge() returns false: soft delete (only update deleted_at field)
	if db.enablePurge == nil {
		db.enablePurge = util.ValueOf(db.m.Purge())
	}

	return nil
}

// WithDB sets the underlying GORM database instance for this database manipulator.
// This allows switching between different database connections or configurations.
// Only supports *gorm.DB type. Returns the same instance if invalid input is provided.
// Example: database.Database[*model.MeetingRoom]().WithDB(mydb).WithTable("meeting_rooms").List(&rooms)
//
// NOTE: If called `WithTable` database option, auto migration will be disabled.
func (db *database[M]) WithDB(x any) types.Database[M] {
	var empty *gorm.DB
	if x == nil || x == new(gorm.DB) || x == empty {
		return db
	}
	// v := reflect.ValueOf(x)
	// if v.Kind() != reflect.Pointer {
	// 	return db
	// }
	// if v.IsNil() {
	// 	return db
	// }
	_db, ok := x.(*gorm.DB)
	if !ok {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithDB")).Warn("invalid database type, expect *gorm.DB")
		return db
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	ctx := db.ins.Statement.Context
	if ctx == nil {
		ctx = context.Background()
		if db.ctx != nil {
			ctx = db.ctx.Context()
		}
	}
	// If "db.shouldAutoMigrate" is not nil, it means the database options `WithTable` was called.
	// If called `WithTable`, "auto migration" must be disabled.
	if db.shouldAutoMigrate == nil {
		// Use database identifier + model type as key to support multiple database instances
		dbIdentifier := getDBIdentifier(_db)
		modelType := reflect.TypeFor[M]().String()
		migrationKey := fmt.Sprintf("%s:%s", dbIdentifier, modelType)
		if _, loaded := migratedModelMap.LoadOrStore(migrationKey, struct{}{}); !loaded {
			flag := new(bool)
			*flag = true
			db.shouldAutoMigrate = flag
		}
	}
	if strings.ToLower(config.App.Logger.Level) == "debug" {
		db.ins = _db.WithContext(ctx).Debug().Limit(defaultLimit)
	} else {
		db.ins = _db.WithContext(ctx).Limit(defaultLimit)
	}
	return db
}

// WithTable sets the table name for database operations, overriding the default table name
// derived from the model type. This is useful for working with custom table names or views.
// Often used in combination with WithDB method.
// Example: database.Database[*model.MeetingRoom]().WithDB(mydb).WithTable("meeting_rooms").List(&rooms)
//
// NOTE: Called this Option menas auto migration will be disabled.
func (db *database[M]) WithTable(name string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.shouldAutoMigrate = new(bool)
	db.tableName = name
	return db
}

// WithTx returns a new database manipulator with transaction context.
// This method allows using an existing transaction to operate on multiple resource types.
// The tx parameter should be a *gorm.DB transaction instance or any compatible transaction type.
// Example:
//
//	database.Database[*User](nil).TransactionFunc(func(tx any) error {
//	    // Use the same transaction for different resource types
//	    if err := database.Database[*User](nil).WithTx(tx).Create(&user); err != nil {
//	        return err
//	    }
//	    if err := database.Database[*Order](nil).WithTx(tx).Create(&order); err != nil {
//	        return err
//	    }
//	    return nil
//	})
func (db *database[M]) WithTx(tx any) types.Database[M] {
	var empty *gorm.DB
	if tx == nil || tx == new(gorm.DB) || tx == empty {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithTx")).Warn("invalid database type, expect *gorm.DB")
		return db
	}

	_tx, ok := tx.(*gorm.DB)
	if !ok || _tx == nil {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithTx")).Warn("invalid database type, expect *gorm.DB")
		return db
	}

	// return &database[M]{
	// 	ins:          _tx.Model(*new(M)),
	// 	ctx:          db.ctx,
	// 	rollbackFunc: db.rollbackFunc,
	// }
	db.ins = _tx.Session(&gorm.Session{
		SkipDefaultTransaction: false,
		NewDB:                  false,
	})

	return db
}

// WithBatchSize sets the batch size for batch operations such as batch insert, update, or delete.
// Controls how many records are processed in a single database operation to optimize performance.
//
// Parameters:
//   - size: The number of records to process per batch. Must be greater than 0.
//     If set to 0 or not called, uses default batch sizes:
//   - Create/Update: 1000 records per batch
//   - Delete: 10000 records per batch
//
// Affected Operations:
//   - Create: Batch inserts records in chunks of the specified size
//   - Update: Batch updates records in chunks of the specified size
//   - Delete: Batch deletes records in chunks of the specified size
//     Note: Delete operations use a separate default (10000) if size is not set
//
// Performance Considerations:
//   - Larger batch sizes improve performance by reducing database round trips
//   - However, larger batches consume more memory and may hit database limits
//   - Recommended range: 100-5000 for most use cases
//   - Very large batches (>10000) may cause memory issues or exceed database limits
//
// Method Chaining:
//
//	WithBatchSize can be chained with other methods and applies to subsequent operations.
//
// Examples:
//
//	// Set batch size for Create operation
//	database.Database[*model.User](nil).WithBatchSize(1000).Create(users...)
//
//	// Set batch size for Update operation
//	database.Database[*model.User](nil).WithBatchSize(500).Update(users...)
//
//	// Set batch size for Delete operation
//	database.Database[*model.User](nil).WithBatchSize(2000).Delete(users...)
//
//	// Combined with other methods
//	database.Database[*model.User](nil).
//	    WithBatchSize(1000).
//	    WithDebug().
//	    Create(users...)
//
// NOTE: If size is 0 or not set, default batch sizes are used (1000 for Create/Update, 10000 for Delete).
// NOTE: The batch size setting persists for the database instance and affects all subsequent operations.
func (db *database[M]) WithBatchSize(size int) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	// db.db = db.db.Session(&gorm.Session{CreateBatchSize: db.batchSize})
	db.batchSize = size
	return db
}

// WithDebug enables debug mode for database operations, showing detailed SQL queries and execution info.
// This setting has higher priority than config.Server.LogLevel and overrides the default value (false).
func (db *database[M]) WithDebug() types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.ins = db.ins.Debug()
	return db
}

// WithIndex specifies database index hints for query optimization.
// The first parameter is the index name, and the second optional parameter specifies the hint type.
// If no hint type is provided, defaults to USE INDEX.
//
// Parameters:
//   - indexName: The name of the index to hint. Empty or whitespace-only names are ignored.
//   - hint: Optional hint mode. If not provided, defaults to consts.IndexHintUse.
//     Supported modes:
//   - consts.IndexHintUse: Suggests the database to use the specified index
//   - consts.IndexHintForce: Forces the database to use the specified index
//   - consts.IndexHintIgnore: Tells the database to ignore the specified index
//
// Supported Query Methods:
//   - List: Query multiple records with index hint
//   - Get: Query a single record by ID with index hint
//   - Count: Count records with index hint
//   - First: Get the first record with index hint
//   - Last: Get the last record with index hint
//   - Take: Get any record with index hint
//
// IMPORTANT: Index hints are ONLY supported in SELECT queries (List, Get, Count, First, Last, Take).
// They are NOT supported in INSERT, UPDATE, DELETE operations. Using WithIndex with Create, Update,
// or Delete methods will result in SQL syntax errors.
//
// Database Compatibility:
//   - MySQL: Fully supported. All hint modes work as expected.
//     If the index doesn't exist, MySQL may return an error.
//   - SQLite/PostgreSQL/Other databases: Not supported.
//     This method will log a warning and skip the hint silently.
//     The query will execute normally without the index hint.
//
// Empty Index Name Handling:
//   - Empty string ("") or whitespace-only strings are automatically trimmed and ignored.
//   - The query will execute normally without any index hint.
//   - This allows safe chaining even when the index name is conditionally provided.
//
// Method Chaining:
//
//	WithIndex can be chained with other query methods like WithQuery, WithSelect, etc.
//	The index hint will be applied to the final query.
//
// Examples:
//
//	// Default USE INDEX hint
//	database.Database[*model.User](nil).WithIndex("idx_name").List(&users)
//
//	// Explicit USE INDEX hint
//	database.Database[*model.User](nil).WithIndex("idx_name", consts.IndexHintUse).List(&users)
//
//	// FORCE INDEX hint
//	database.Database[*model.User](nil).WithIndex("idx_name", consts.IndexHintForce).List(&users)
//
//	// IGNORE INDEX hint
//	database.Database[*model.User](nil).WithIndex("idx_name", consts.IndexHintIgnore).List(&users)
//
//	// With Get method
//	var user *model.User
//	database.Database[*model.User](nil).WithIndex("idx_name").Get(user, userID)
//
//	// With Count method
//	var count int64
//	database.Database[*model.User](nil).WithIndex("idx_name").Count(&count)
//
//	// Combined with WithQuery
//	database.Database[*model.User](nil).
//	    WithIndex("idx_name").
//	    WithQuery(&model.User{Name: "John"}).
//	    List(&users)
//
//	// Empty index name (ignored, query works normally)
//	database.Database[*model.User](nil).WithIndex("").List(&users)
//	database.Database[*model.User](nil).WithIndex("   ").List(&users)
//
// NOTE: Index hints are MySQL-specific. On other databases, the hint is silently ignored.
// NOTE: Empty or whitespace-only index names are automatically ignored for safe chaining.
// NOTE: Unknown hint modes will default to USE INDEX with a warning logged.
func (db *database[M]) WithIndex(indexName string, hint ...consts.IndexHintMode) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Trim whitespace from the index name
	indexName = strings.TrimSpace(indexName)
	if len(indexName) == 0 {
		return db
	}

	// Check if database supports index hints (only MySQL supports them)
	// SQLite, PostgreSQL, and other databases don't support index hints
	if db.ins == nil {
		return db
	}

	// Get database driver name to check if it's MySQL
	driverName := db.ins.Name()
	if driverName != "mysql" {
		// Index hints are only supported by MySQL
		// For other databases (SQLite, PostgreSQL, etc.), log a warning and skip
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithIndex")).Warnf(
			"index hints are not supported by %s database, skipping index hint for: %s",
			driverName, indexName,
		)
		return db
	}

	// Determine the hint type, default to USE if not provided
	var hintMode consts.IndexHintMode
	if len(hint) > 0 {
		hintMode = hint[0]
	} else {
		hintMode = consts.IndexHintUse
	}

	// Apply the appropriate hint
	switch hintMode {
	case consts.IndexHintUse:
		db.ins = db.ins.Clauses(hints.UseIndex(indexName))
	case consts.IndexHintForce:
		db.ins = db.ins.Clauses(hints.ForceIndex(indexName))
	case consts.IndexHintIgnore:
		db.ins = db.ins.Clauses(hints.IgnoreIndex(indexName))
	default:
		logger.Database.Warnf(`unknown index hint mode: %s, using "USE INDEX"`, hintMode)
		// Default to USE INDEX for unknown modes
		db.ins = db.ins.Clauses(hints.UseIndex(indexName))
	}

	return db
}

// WithQuery sets query conditions based on the provided model struct fields.
// It supports exact matching, fuzzy matching (LIKE/REGEXP), OR/AND logic, and raw SQL queries.
// Non-zero fields in the model will be used as query conditions.
//
// Parameters:
//   - query: A model instance with fields set as query conditions. Can be nil to indicate empty query.
//     When nil or all fields are zero values, it's treated as an empty query.
//     Supported field types: string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool, pointer types.
//   - config: Optional QueryConfig to control query behavior (fuzzy matching, empty queries, OR logic, raw SQL)
//
// Query Behavior:
//
//	Exact Match (Default):
//	- Single value: Uses IN clause with one value (WHERE name IN ('John'))
//	- Multiple values (comma-separated): Uses IN clause with multiple values (WHERE name IN ('John', 'Jack'))
//	- Multiple fields: Uses AND logic to combine conditions (WHERE name IN ('John') AND age IN (18))
//	- Empty strings in comma-separated values are automatically skipped
//
//	FuzzyMatch:
//	- Single value: Uses LIKE pattern (WHERE name LIKE '%John%')
//	- Multiple values (comma-separated): Uses REGEXP pattern (WHERE name REGEXP '.*John.*|.*Jack.*')
//	- REGEXP special characters are automatically escaped using regexp.QuoteMeta
//	- Empty strings in comma-separated values are automatically skipped to prevent matching all records
//	- Note: REGEXP may not be available in all databases (e.g., SQLite requires extension)
//
//	UseOr:
//	- When true: Combines multiple field conditions with OR instead of AND
//	- First condition always uses WHERE, subsequent conditions use OR
//	- Example: WHERE name IN ('John') OR email IN ('john@example.com')
//	- Works with both exact match and fuzzy match
//
//	RawQuery:
//	- When provided, model fields are completely ignored and only RawQuery is used
//	- Works even when query is nil
//	- Supports parameterized queries with RawQueryArgs
//	- Example: WHERE age > ? AND status = ?
//
//	AllowEmpty:
//	- By default (false): Empty queries are blocked for safety (adds WHERE 1 = 0)
//	- When true: Allows empty queries to match all records (full table scan)
//	- Empty query cases: nil, empty struct, all fields are zero values, all field values are empty strings
//	- Critical: Use with caution, especially for Delete operations
//
// Examples:
//
//	// Exact match - single field, single value
//	WithQuery(&model.User{Name: "John"})  // WHERE name IN ('John')
//
//	// Exact match - single field, multiple values (comma-separated)
//	WithQuery(&model.User{Name: "John,Jack"})  // WHERE name IN ('John', 'Jack')
//	WithQuery(&model.User{ID: "id1,id2,id3"})  // WHERE id IN ('id1', 'id2', 'id3')
//
//	// Exact match - multiple fields (AND logic)
//	WithQuery(&model.User{Name: "John", Age: 18})  // WHERE name IN ('John') AND age IN (18)
//	WithQuery(&model.User{Name: "John", Age: 18, Email: "john@example.com"})  // WHERE name IN ('John') AND age IN (18) AND email IN ('john@example.com')
//
//	// Fuzzy match - single value (LIKE)
//	WithQuery(&model.User{Name: "John"}, types.QueryConfig{FuzzyMatch: true})  // WHERE name LIKE '%John%'
//
//	// Fuzzy match - multiple values (REGEXP)
//	WithQuery(&model.User{Name: "John,Jack"}, types.QueryConfig{FuzzyMatch: true})  // WHERE name REGEXP '.*John.*|.*Jack.*'
//
//	// Fuzzy match - empty strings in comma-separated values are skipped
//	WithQuery(&model.User{Name: "John,,Jack"}, types.QueryConfig{FuzzyMatch: true})  // WHERE name REGEXP '.*John.*|.*Jack.*'
//
//	// OR logic to combine conditions
//	WithQuery(&model.User{Name: "John", Email: "john@example.com"}, types.QueryConfig{UseOr: true})
//	// WHERE name IN ('John') OR email IN ('john@example.com')
//
//	// OR logic with fuzzy match
//	WithQuery(&model.User{Name: "John", Email: "example"}, types.QueryConfig{UseOr: true, FuzzyMatch: true})
//	// WHERE name LIKE '%John%' OR email LIKE '%example%'
//
//	// Raw SQL query (model fields are ignored)
//	WithQuery(&model.User{}, types.QueryConfig{RawQuery: "age > ? AND status = ?", RawQueryArgs: []any{18, "active"}})
//	WithQuery(nil, types.QueryConfig{RawQuery: "created_at BETWEEN ? AND ?", RawQueryArgs: []any{startDate, endDate}})
//	WithQuery(&model.User{Name: "John"}, types.QueryConfig{RawQuery: "age > ?", RawQueryArgs: []any{18}})  // Name is ignored
//
//	// Empty query (blocked by default for safety)
//	WithQuery(nil)  // WHERE 1 = 0 (returns no records)
//	WithQuery(&model.User{})  // WHERE 1 = 0 (returns no records)
//	WithQuery(&model.User{Name: "", Email: ""})  // WHERE 1 = 0 (all values are empty)
//
//	// Empty query with AllowEmpty=true (returns all records)
//	WithQuery(nil, types.QueryConfig{AllowEmpty: true})  // Returns all records
//	WithQuery(&model.User{}, types.QueryConfig{AllowEmpty: true})  // Returns all records
//
//	// Query with some empty and some non-empty fields (works normally)
//	WithQuery(&model.User{Name: "John", Email: ""})  // WHERE name IN ('John') (Email is ignored)
//
//	// Combined options
//	WithQuery(&model.User{Name: "John"}, types.QueryConfig{
//	    FuzzyMatch: true,
//	    UseOr:      true,
//	    AllowEmpty: false,
//	})
//
// NOTE: The underlying type must be pointer to struct, otherwise panic will occur.
// NOTE: Empty query conditions (nil or zero value) are blocked by default for safety to prevent
//
//	catastrophic data loss (e.g., deleting all records). Use QueryConfig{AllowEmpty: true} to override.
//
// NOTE: When RawQuery is provided, all model fields are ignored regardless of their values.
// NOTE: REGEXP function may not be available in all databases (e.g., SQLite requires extension).
//
//	For SQLite compatibility, consider using FuzzyMatch with single values (LIKE) or RawQuery.
func (db *database[M]) WithQuery(query M, config ...types.QueryConfig) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Parse query configuration
	var cfg types.QueryConfig
	if len(config) > 0 {
		cfg = config[0]
	}
	// cfg.FuzzyMatch: default false (exact match)
	// cfg.AllowEmpty: default false (block empty queries for safety)

	queryVal := reflect.ValueOf(query)
	// Handle RawQuery first (works even if query is nil)
	// When RawQuery is provided, model fields are ignored as per documentation
	if len(cfg.RawQuery) > 0 {
		db.ins = db.ins.Where(cfg.RawQuery, cfg.RawQueryArgs...)
		// If RawQuery is provided, ignore model fields and return early
		// RawQuery alone is sufficient for query conditions
		return db
	}

	// Check if query is nil or empty
	var empty M
	if queryVal.IsNil() || reflect.DeepEqual(query, empty) {
		// Treat nil/empty as empty query
		// Note: RawQuery is already handled above if provided
		if !cfg.AllowEmpty {
			logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Warn("query is nil or empty, adding safety condition to prevent matching all records")
			db.ins = db.ins.Where("1 = 0")
			return db
		}
		// AllowEmpty=true: allow matching all records
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Info("query is nil or empty but AllowEmpty=true, allowing full table scan")
		return db
	}

	// Process non-nil, non-empty query
	typ := reflect.TypeOf(query).Elem()
	val := reflect.ValueOf(query).Elem()
	q := make(map[string]string)

	// for i := 0; i < typ.NumField(); i++ {
	// 	// fmt.Println("---------------- for type ", typ.Field(i).Type, typ.Field(i).Type.Kind(), typ.Field(i).Name, val.Field(i).IsZero())
	// 	if val.Field(i).IsZero() {
	// 		continue
	// 	}
	//
	// 	switch typ.Field(i).Type.Kind() {
	// 	case reflect.Chan, reflect.Map, reflect.Func:
	// 		continue
	// 	case reflect.Struct:
	// 		// All `model.XXX` extends the basic model named `Base`,
	// 		if typ.Field(i).Name == "Base" {
	// 			if !val.Field(i).FieldByName("ID").IsZero() {
	// 				// Not overwrite the "ID" value set in types.Model.
	// 				// The "ID" value set in types.Model has higher priority than base model.
	// 				if _, loaded := q["id"]; !loaded {
	// 					q["id"] = val.Field(i).FieldByName("ID").Interface().(string)
	// 				}
	// 			}
	// 		} else {
	// 			structFieldToMap(typ.Field(i).Type, val.Field(i), q)
	// 		}
	// 		continue
	// 	}
	// 	// "json" tag priority is higher than typ.Field(i).Name
	// 	jsonTagStr := typ.Field(i).Tag.Get("json")
	// 	jsonTagItems := strings.Split(jsonTagStr, ",")
	// 	// NOTE: strings.Split always returns at least one element(empty string)
	// 	// We should not use len(jsonTagItems) to check the json tags exists.
	// 	jsonTag := ""
	// 	if len(jsonTagItems) == 0 {
	// 		// the structure lowercase field name as the query condition.
	// 		jsonTagItems[0] = typ.Field(i).Name
	// 	}
	// 	jsonTag = jsonTagItems[0]
	// 	if len(jsonTag) == 0 {
	// 		// the structure lowercase field name as the query condition.
	// 		jsonTag = typ.Field(i).Name
	// 	}
	// 	// "schema" tag have higher priority than "json" tag
	// 	schemaTagStr := strings.TrimSpace(typ.Field(i).Tag.Get("schema"))
	// 	if len(schemaTagStr) > 0 && schemaTagStr != jsonTagStr {
	// 		fmt.Println("-------------  schema tag overwrite json tag")
	// 		jsonTag = strings.TrimSpace(typ.Field(i).Tag.Get("schema"))
	// 	}
	//
	// 	v := val.Field(i).Interface()
	// 	var _v string
	// 	switch val.Field(i).Kind() {
	// 	case reflect.Bool:
	// 		// 由于 WHERE IN 语句会自动加上单引号,比如 WHERE `default` IN ('true')
	// 		// 但是我们想要的是 WHERE `default` IN (true),
	// 		// 所以没办法就只能直接转成 int 了.
	// 		_v = fmt.Sprintf("%d", boolToInt(v.(bool)))
	// 	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	// 		_v = fmt.Sprintf("%d", v)
	// 	case reflect.Float32, reflect.Float64:
	// 		_v = fmt.Sprintf("%g", v)
	// 	case reflect.String:
	// 		_v = fmt.Sprintf("%s", v)
	// 	case reflect.Pointer:
	// 		v = val.Field(i).Elem().Interface()
	// 		// switch typ.Elem().Kind() {
	// 		switch val.Field(i).Elem().Kind() {
	// 		case reflect.Bool:
	// 			_v = fmt.Sprintf("%d", boolToInt(v.(bool)))
	// 		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	// 			_v = fmt.Sprintf("%d", v)
	// 		case reflect.Float32, reflect.Float64:
	// 			_v = fmt.Sprintf("%g", v)
	// 		case reflect.String:
	// 			_v = fmt.Sprintf("%s", v)
	// 		case reflect.Struct, reflect.Map, reflect.Chan, reflect.Func: // ignore the struct, map, chan, func
	// 		default:
	// 			_v = fmt.Sprintf("%v", v)
	// 		}
	// 	case reflect.Slice:
	// 		_len := val.Field(i).Len()
	// 		if _len == 0 {
	// 			logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Warn("reflect.Slice length is 0")
	// 			_len = 1
	// 		}
	// 		slice := reflect.MakeSlice(val.Field(i).Type(), _len, _len)
	// 		// fmt.Println("--------------- slice element", slice.Index(0), slice.Index(0).Kind(), slice.Index(0).Type())
	// 		switch slice.Index(0).Kind() {
	// 		case reflect.String: // handle string slice.
	// 			// WARN: val.Field(i).Type() is model.GormStrings not []string,
	// 			// execute statement `slice.Interface().([]string)` directly will case panic.
	// 			// _v = strings.Join(slice.Interface().([]string), ",") // the slice type is GormStrings not []string.
	// 			// We should make the slice of []string again.
	// 			slice = reflect.MakeSlice(reflect.TypeOf([]string{}), _len, _len)
	// 			reflect.Copy(slice, val.Field(i))
	// 			_v = strings.Join(slice.Interface().([]string), ",")
	// 		default:
	// 			_v = fmt.Sprintf("%v", v)
	// 		}
	// 	case reflect.Struct, reflect.Map, reflect.Chan, reflect.Func: // ignore the struct, map, chan, func
	// 	default:
	// 		_v = fmt.Sprintf("%v", v)
	// 	}
	//
	// 	// json tag name naming format must be same as gorm table columns,
	// 	// both should be "snake case" or "camel case".
	// 	// gorm table columns naming format default to 'snake case',
	// 	// so the json tag name is converted to "snake case here"
	// 	// q[strcase.SnakeCase(jsonTag)] = val.Field(i).Interface()
	// 	q[strcase.SnakeCase(jsonTag)] = _v
	// }

	structFieldToMap(db.ctx, typ, val, q)
	// fmt.Println("------------- WithQuery", q)

	// CRITICAL SAFETY CHECK: Empty query conditions
	//
	// Empty query will match ALL records, which is dangerous when:
	// 1. The result is used for subsequent Delete operations → deletes all data (CATASTROPHIC!)
	// 2. Large datasets returned without pagination → performance/memory issues
	//
	// Empty Query Examples:
	//   - WithQuery(nil)                         → nil query
	//   - WithQuery(&User{})                    → all fields are zero values
	//   - WithQuery(&User{Name: "", Email: ""}) → all field values are empty strings
	//   - WithQuery(&KV{Key: ""})               → happens when removed slice is empty
	//
	// By default, empty queries (nil or zero value) are blocked by adding "WHERE 1 = 0" condition.
	// To allow empty queries, use: WithQuery(nil, QueryConfig{AllowEmpty: true}) or
	//                              WithQuery(&User{}, QueryConfig{AllowEmpty: true})
	if len(q) == 0 {
		if !cfg.AllowEmpty {
			logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Warn("all query fields are empty, adding safety condition to prevent matching all records")
			db.ins = db.ins.Where("1 = 0")
			return db
		}
		// AllowEmpty=true: allow matching all records
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Info("all query fields are empty but AllowEmpty=true, allowing full table scan")
		return db
	}

	if cfg.FuzzyMatch {
		// // Deprecated!
		// for k, v := range q {
		// 	// WARN: THE SQL STATEMENT MUST CONTAINS backticks ``.
		// 	db.db = db.db.Where(fmt.Sprintf("`%s` LIKE ?", k), fmt.Sprintf("%%%v%%", v))
		// }

		// If the query strings has multiple value(separated by ',')
		// construct the 'WHERE' 'REGEXP' SQL statement
		// eg: SELECT * FROM `assets` WHERE `category_level2_id` REGEXP '.*XS.*|.*NU.*'
		//     SELECT count(*) FROM `assets` WHERE `category_level2_id` REGEXP '.*XS.*|.*NU.*'
		hasValidCondition := false
		isFirstCondition := true
		for k, v := range q {
			items := strings.Split(v, ",")
			// skip the string slice which all element is empty.
			if len(strings.Join(items, "")) == 0 {
				continue
			}
			hasValidCondition = true
			if len(items) > 1 { // If the query string has multiple value(separated by ','), using regexp
				var regexpVal string
				for _, item := range items {
					// Skip empty items to avoid matching all records (.*.* pattern)
					if len(item) == 0 {
						continue
					}
					// WARN: not forget to escape the regexp value using regexp.QuoteMeta.
					// eg: localhost\hello.world -> localhost\\hello\.world
					regexpVal = regexpVal + "|.*" + regexp.QuoteMeta(item) + ".*"
				}
				// If all items were empty after filtering, skip this condition
				if len(regexpVal) == 0 {
					continue
				}
				regexpVal = strings.TrimPrefix(regexpVal, "|")
				// db.db = db.db.Where(fmt.Sprintf("`%s` REGEXP ?", k), regexpVal)
				if cfg.UseOr && !isFirstCondition {
					db.ins = db.ins.Or(fmt.Sprintf("`%s` REGEXP ?", k), regexpVal)
				} else {
					db.ins = db.ins.Where(fmt.Sprintf("`%s` REGEXP ?", k), regexpVal)
				}
			} else { // If the query string has only one value, using LIKE
				// db.db = db.db.Where(fmt.Sprintf("`%s` LIKE ?", k), fmt.Sprintf("%%%v%%", v))
				if cfg.UseOr && !isFirstCondition {
					db.ins = db.ins.Or(fmt.Sprintf("`%s` LIKE ?", k), fmt.Sprintf("%%%v%%", v))
				} else {
					db.ins = db.ins.Where(fmt.Sprintf("`%s` LIKE ?", k), fmt.Sprintf("%%%v%%", v))
				}
			}
			isFirstCondition = false
		}
		// CRITICAL: Check if all query values are empty after filtering
		// Even if query map is not empty, all values might be empty strings
		// Example: &User{Name: "", Email: ""} has fields but all values are empty
		if !hasValidCondition {
			if !cfg.AllowEmpty {
				logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Warn("all query values are empty, adding safety condition to prevent matching all records")
				db.ins = db.ins.Where("1 = 0")
			} else {
				logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Info("all query values are empty but AllowEmpty=true, allowing full table scan")
			}
		}
	} else {
		// // Deprecated!
		// // SELECT * FROM `assets` WHERE `assets`.`category_level2_id` = 'NU
		// // SELECT count(*) FROM `assets` WHERE `assets`.`category_level2_id` = 'NU'
		// db.db = db.db.Where(query)

		// If the query string has multiple value(separated by ','),
		// construct the 'WHERE' 'IN' SQL statement.
		// eg: SELECT id FROM users WHERE name IN ('user01', 'user02', 'user03', 'user04')
		hasValidCondition := false
		isFirstCondition := true
		for k, v := range q {
			items := strings.Split(v, ",")
			if len(strings.Join(items, "")) == 0 {
				continue
			}
			hasValidCondition = true
			// db.db = db.db.Where(fmt.Sprintf("`%s` IN (?)", k), items)
			if cfg.UseOr && !isFirstCondition {
				db.ins = db.ins.Or(fmt.Sprintf("`%s` IN (?)", k), items)
			} else {
				db.ins = db.ins.Where(fmt.Sprintf("`%s` IN (?)", k), items)
			}
			isFirstCondition = false
		}
		// CRITICAL: Check if all query values are empty after filtering
		// Even if query map is not empty, all values might be empty strings
		// Example: &User{Name: "", Email: ""} has fields but all values are empty
		if !hasValidCondition {
			if !cfg.AllowEmpty {
				logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Warn("all query values are empty, adding safety condition to prevent matching all records")
				db.ins = db.ins.Where("1 = 0")
			} else {
				logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithQuery")).Info("all query values are empty but AllowEmpty=true, allowing full table scan")
			}
		}
	}
	return db
}

// structFieldToMap extracts the field tags from a struct and writes them into a map.
// This map can then be used to build SQL query conditions.
// FIXME: if the field type is boolean or ineger, disable the fuzzy matching.
func structFieldToMap(ctx *types.DatabaseContext, typ reflect.Type, val reflect.Value, q map[string]string) {
	if q == nil {
		q = make(map[string]string)
	}
	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldTyp := field.Type
		fieldVal := val.Field(i)

		if fieldVal.IsZero() {
			continue
		}
		if !fieldVal.CanInterface() {
			continue
		}
		fieldTyp, fieldVal, ok := indirectTypeAndValue(fieldTyp, fieldVal)
		if !ok {
			continue
		}

		switch fieldTyp.Kind() {
		case reflect.Chan, reflect.Map, reflect.Func:
			continue
		case reflect.Struct:
			// All `model.XXX` extends the basic model named `Base`,
			if field.Name == "Base" {
				if !fieldVal.FieldByName("CreatedBy").IsZero() {
					// Not overwrite the "CreatedBy" value set in types.Model.
					// The "CreatedBy" value set in types.Model has higher priority than base model.
					if _, loaded := q["created_by"]; !loaded {
						q["created_by"] = fieldVal.FieldByName("CreatedBy").Interface().(string) //nolint:errcheck
					}
				}
				if !fieldVal.FieldByName("UpdatedBy").IsZero() {
					// Not overwrite the "UpdatedBy" value set in types.Model.
					// The "UpdatedBy" value set in types.Model has higher priority than base model.
					if _, loaded := q["updated_by"]; !loaded {
						q["updated_by"] = fieldVal.FieldByName("UpdatedBy").Interface().(string) //nolint:errcheck
					}
				}
				if !fieldVal.FieldByName("ID").IsZero() {
					// Not overwrite the "ID" value set in types.Model.
					// The "ID" value set in types.Model has higher priority than base model.
					if _, loaded := q["id"]; !loaded {
						q["id"] = fieldVal.FieldByName("ID").Interface().(string) //nolint:errcheck
					}
				}
			} else {
				structFieldToMap(ctx, fieldTyp, fieldVal, q)
			}
			continue
		}
		// "json" tag priority is higher than field.Name
		jsonTagStr := strings.TrimSpace(field.Tag.Get("json"))
		jsonTagItems := strings.Split(jsonTagStr, ",")
		// NOTE: strings.Split always returns at least one element(empty string)
		// We should not use len(jsonTagItems) to check the json tags exists.
		var jsonTag string
		if len(jsonTagItems) == 0 {
			// the structure lowercase field name as the query condition.
			jsonTagItems[0] = field.Name
		}
		jsonTag = jsonTagItems[0]
		if len(jsonTag) == 0 {
			// the structure lowercase field name as the query condition.
			jsonTag = field.Name
		}
		// "schema" tag have higher priority than "json" tag
		schemaTagStr := strings.TrimSpace(field.Tag.Get("schema"))
		schemaTagItems := strings.Split(schemaTagStr, ",")
		schemaTag := ""
		if len(schemaTagItems) > 0 {
			schemaTag = schemaTagItems[0]
		}
		if len(schemaTag) > 0 && schemaTag != jsonTag {
			fmt.Printf("------------------ json tag replace by schema tag: %s -> %s\n", jsonTag, schemaTag)
			jsonTag = schemaTag
		}

		if !fieldVal.CanInterface() {
			continue
		}
		v := fieldVal.Interface()
		var _v string
		switch fieldVal.Kind() {
		case reflect.Bool:
			// 由于 WHERE IN 语句会自动加上单引号,比如 WHERE `default` IN ('true')
			// 但是我们想要的是 WHERE `default` IN (true),
			// 所以没办法就只能直接转成 int 了.
			_v = fmt.Sprintf("%d", boolToInt(v.(bool))) //nolint:errcheck
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			_v = fmt.Sprintf("%d", v)
		case reflect.Float32, reflect.Float64:
			_v = fmt.Sprintf("%g", v)
		case reflect.String:
			_v = fmt.Sprintf("%s", v)
		case reflect.Pointer:
			v = fieldVal.Elem().Interface()
			// switch typ.Elem().Kind() {
			switch fieldVal.Elem().Kind() {
			case reflect.Bool:
				_v = fmt.Sprintf("%d", boolToInt(v.(bool))) //nolint:errcheck
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				_v = fmt.Sprintf("%d", v)
			case reflect.Float32, reflect.Float64:
				_v = fmt.Sprintf("%g", v)
			case reflect.String:
				_v = fmt.Sprintf("%s", v)
			case reflect.Struct, reflect.Map, reflect.Chan, reflect.Func: // ignore the struct, map, chan, func
			default:
				_v = fmt.Sprintf("%v", v)
			}
		case reflect.Slice:
			_len := fieldVal.Len()
			if _len == 0 {
				logger.Database.WithDatabaseContext(ctx, consts.Phase("WithQuery")).Warn("reflect.Slice length is 0")
				_len = 1
			}
			slice := reflect.MakeSlice(fieldVal.Type(), _len, _len)
			// fmt.Println("--------------- slice element", slice.Index(0), slice.Index(0).Kind(), slice.Index(0).Type())
			switch slice.Index(0).Kind() {
			case reflect.String: // handle string slice.
				// WARN: fieldVal.Type() is model.GormStrings not []string,
				// execute statement `slice.Interface().([]string)` directly will case panic.
				// _v = strings.Join(slice.Interface().([]string), ",") // the slice type is GormStrings not []string.
				// We should make the slice of []string again.
				slice = reflect.MakeSlice(reflect.TypeFor[[]string](), _len, _len)
				reflect.Copy(slice, fieldVal)
				_v = strings.Join(slice.Interface().([]string), ",") //nolint:errcheck
			default:
				_v = fmt.Sprintf("%v", v)
			}
		case reflect.Struct, reflect.Map, reflect.Chan, reflect.Func: // ignore the struct, map, chan, func
		default:
			_v = fmt.Sprintf("%v", v)
		}

		// json tag name naming format must be same as gorm table columns,
		// both should be "snake case" or "camel case".
		// gorm table columns naming format default to 'snake case',
		// so the json tag name is converted to "snake case here"
		// q[strcase.SnakeCase(jsonTag)] = fieldVal.Interface()
		q[strcase.SnakeCase(jsonTag)] = _v
	}
}

// WithCursor enables cursor-based pagination.
// cursorValue is the value of the last record in the previous page.
// next indicates the direction of pagination:
//   - true: fetch records after the cursor (next page)
//   - false: fetch records before the cursor (previous page)
//
// Example:
//
//	// First page (no cursor)
//	db.Database[*model.User]().WithLimit(10).List(&users)
//	// Next page (using last user's ID as cursor)
//	lastID := users[len(users)-1].ID
//	db.Database[*model.User]().WithCursor(lastID, true).WithLimit(10).List(&nextUsers)
//	db.Database[*model.User]().WithCursor(lastID, true, "created_at").WithLimit(10).List(&nextUsers)
func (db *database[M]) WithCursor(cursorValue string, next bool, fields ...string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()

	if len(cursorValue) == 0 {
		return db
	}

	db.enableCursor = true
	db.cursorValue = cursorValue
	db.cursorNext = next

	// TODO: support multiple cursor fields
	if len(fields) > 0 {
		db.cursorField = fields[0]
	}
	// Default cursor field is "id" if not specified
	if db.cursorField == "" {
		db.cursorField = "id"
	}

	return db
}

// applyCursorPagination applies cursor-based pagination to the query if cursor is set.
func (db *database[M]) applyCursorPagination() {
	if db.enableCursor {
		// Apply cursor condition based on direction
		if db.cursorNext {
			// Next page: get records after the cursor
			db.ins = db.ins.Where(fmt.Sprintf("`%s` > ?", db.cursorField), db.cursorValue)
			// Order by cursor field ascending for next page
			db.ins = db.ins.Order(fmt.Sprintf("`%s` ASC", db.cursorField))
		} else {
			// Previous page: get records before the cursor
			db.ins = db.ins.Where(fmt.Sprintf("`%s` < ?", db.cursorField), db.cursorValue)
			// Order by cursor field descending for previous page
			db.ins = db.ins.Order(fmt.Sprintf("`%s` DESC", db.cursorField))
		}
	}
}

// WithTimeRange filters records within a specific time range.
// Supports flexible time range queries:
//   - Both times provided: uses BETWEEN clause
//   - Only startTime provided (endTime is zero): uses >= clause
//   - Only endTime provided (startTime is zero): uses <= clause
//   - Both times are zero: returns without filtering
//
// Parameters:
//   - columnName: The name of the time column to filter on
//   - startTime: The start time of the range (inclusive). Use zero value to ignore.
//   - endTime: The end time of the range (inclusive). Use zero value to ignore.
//
// Examples:
//
//	// Range query: created_at BETWEEN start AND end
//	WithTimeRange("created_at", time.Now().AddDate(0, -1, 0), time.Now())
//
//	// After query: created_at >= start
//	WithTimeRange("created_at", time.Now().AddDate(0, -1, 0), time.Time{})
//
//	// Before query: created_at <= end
//	WithTimeRange("created_at", time.Time{}, time.Now())
func (db *database[M]) WithTimeRange(columnName string, startTime time.Time, endTime time.Time) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(columnName) == 0 {
		return db
	}

	startIsZero := startTime.IsZero()
	endIsZero := endTime.IsZero()

	// Both times are zero, no filtering
	if startIsZero && endIsZero {
		return db
	}

	// Both times provided, use BETWEEN
	if !startIsZero && !endIsZero {
		db.ins = db.ins.Where(fmt.Sprintf("`%s` BETWEEN ? AND ?", columnName), startTime, endTime)
		return db
	}

	// Only start time provided, use >=
	if !startIsZero && endIsZero {
		db.ins = db.ins.Where(fmt.Sprintf("`%s` >= ?", columnName), startTime)
		return db
	}

	// Only end time provided, use <=
	if startIsZero && !endIsZero {
		db.ins = db.ins.Where(fmt.Sprintf("`%s` <= ?", columnName), endTime)
		return db
	}

	return db
}

// WithSelect specifies fields to select when querying, creating, or updating records.
// The method automatically includes defaultsColumns (id, created_by, updated_by, created_at, updated_at, deleted_at)
// in addition to the specified columns to ensure essential fields are always available.
// Empty or whitespace-only column names are filtered out, and duplicate defaultsColumns are avoided.
//
// Parameters:
//   - columns: Field names to select (defaultsColumns will be automatically added)
//     If no columns are provided, only defaultsColumns will be selected
//
// Returns the same instance if no valid columns are provided after filtering.
//
// WARNING: Using WithSelect may result in the removal of certain fields from table records
// if there are multiple hooks in the service and model layers. Use with caution.
func (db *database[M]) WithSelect(columns ...string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(columns) == 0 {
		db.ins = db.ins.Select(defaultsColumns)
		return db
	}
	_columns := make([]string, 0)
	for i := range columns {
		col := strings.TrimSpace(columns[i])
		if len(col) > 0 && !contains(defaultsColumns, col) {
			_columns = append(_columns, col)
		}
	}
	if len(_columns) == 0 {
		return db
	}
	db.ins = db.ins.Select(append(_columns, defaultsColumns...))
	return db
}

// WithSelectRaw allows specifying raw SQL SELECT clause with optional arguments.
// Unlike WithSelect, this method does not automatically add defaultsColumns.
// Use this when you need full control over the SELECT statement.
//
// Parameters:
//   - query: Raw SQL SELECT clause or column expressions
//   - args: Optional arguments for parameterized queries
//
// Example:
//
//	WithSelectRaw("COUNT(*) as total, AVG(price) as avg_price")
//	WithSelectRaw("users.name, orders.amount WHERE orders.status = ?", "completed")
//
// WithSelectRaw
func (db *database[M]) WithSelectRaw(query any, args ...any) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.ins = db.ins.Select(query, args...)
	return db
}

// WithRollback sets a rollback function for manual transaction control.
// This method is used with TransactionFunc to enable manual rollback control.
// To trigger a manual rollback, call the rollback function directly and return ErrManualRollback.
//
// Example:
//
//	var rollbackFunc func() error
//	err := db.WithRollback(func() error {
//	    // custom rollback logic
//	    return nil
//	}).TransactionFunc(func(tx any) error {
//	    userDB := database.Database[*model.User](nil)
//	    // Get the rollback function from the transaction context
//	    if txDB, ok := userDB.WithTx(tx).(*database[*model.User]); ok && txDB.rollbackFunc != nil {
//	        rollbackFunc = txDB.rollbackFunc
//	    }
//
//	    if err := userDB.WithTx(tx).Create(&user); err != nil {
//	        return err // automatic rollback
//	    }
//
//	    if someCondition {
//	        if rollbackFunc != nil {
//	            rollbackFunc() // execute custom rollback logic
//	        }
//	        return ErrManualRollback // trigger transaction rollback
//	    }
//	    return nil // automatic commit
//	})
func (db *database[M]) WithRollback(rollbackFunc func() error) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.rollbackFunc = rollbackFunc
	return db
}

// WithLock adds row-level locking to the query for concurrent access control.
// Uses SELECT ... FOR UPDATE to prevent other transactions from modifying selected rows.
// Should be used within a transaction to be effective.
//
// Example:
//
//	DB.Transaction(func(tx *gorm.DB) error {
//	    return Database[*User]().
//	        WithLock().
//	        Get(&user, userID)
//	})
//
// WithLock adds locking clause to SELECT statement.
// It must be used within a transaction.
//
// Lock modes:
//   - consts.LockUpdate (default): SELECT ... FOR UPDATE
//   - consts.LockShare: SELECT ... FOR SHARE
//   - consts.LockUpdateNoWait: SELECT ... FOR UPDATE NOWAIT
//   - consts.LockShareNoWait: SELECT ... FOR SHARE NOWAIT
//   - consts.LockUpdateSkipLocked: SELECT ... FOR UPDATE SKIP LOCKED
//   - consts.LockShareSkipLocked: SELECT ... FOR SHARE SKIP LOCKED
//
// Example:
//
//	DB.Transaction(func(tx *gorm.DB) error {
//	    // Default FOR UPDATE lock
//	    err := Database[*Order]().
//	        WithTx(tx).
//	        WithLock().
//	        Get(&order, orderID)
//
//	    // FOR UPDATE NOWAIT
//	    err = Database[*Order]().
//	        WithTx(tx).
//	        WithLock(consts.LockUpdateNoWait).
//	        Get(&order, orderID)
//	})
func (db *database[M]) WithLock(mode ...consts.LockMode) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()

	strength := "UPDATE"
	options := ""
	if len(mode) > 0 {
		switch mode[0] {
		case consts.LockShare:
			strength = "SHARE"
		case consts.LockUpdateNoWait:
			strength = "UPDATE"
			options = "NOWAIT"
		case consts.LockShareNoWait:
			strength = "SHARE"
			options = "NOWAIT"
		case consts.LockUpdateSkipLocked:
			strength = "UPDATE"
			options = "SKIP LOCKED"
		case consts.LockShareSkipLocked:
			strength = "SHARE"
			options = "SKIP LOCKED"
		}
	}

	db.ins = db.ins.Clauses(clause.Locking{
		Strength: strength,
		Options:  options,
	})
	return db
}

// WithJoinRaw adds a raw SQL JOIN clause to the query.
// Provides full control over JOIN operations including INNER, LEFT, RIGHT, and FULL OUTER joins.
//
// Parameters:
//   - query: Raw SQL JOIN clause
//   - args: Optional arguments for parameterized queries
//
// Example:
//
//	WithJoinRaw("LEFT JOIN orders ON users.id = orders.user_id")
//	WithJoinRaw("INNER JOIN categories c ON products.category_id = c.id AND c.status = ?", "active")
//
// WithJoinRaw adds JOIN clause to query.
//
// Basic Join:
//
//	db.WithJoinRaw("JOIN users ON users.id = orders.user_id")
//
// Left Join with conditions:
//
//	db.WithJoinRaw("LEFT JOIN users ON users.id = orders.user_id AND users.active = ?", 1)
//
// Multiple Joins:
//
//	db.WithJoinRaw("LEFT JOIN users ON users.id = orders.user_id").
//	    WithJoinRaw("LEFT JOIN products ON products.id = orders.product_id")
//
// Join with Select:
//
//	db.WithSelectRaw("orders.*, users.name").
//	    WithJoinRaw("LEFT JOIN users ON users.id = orders.user_id")
//
// Complex Examples:
//
// 1. Query order with user info:
//
//	type Order struct {
//	    ID     string `gorm:"primarykey"`
//	    UserID string
//	    Amount float64
//	    User   User   `gorm:"foreignKey:UserID"`
//	}
//
//	type User struct {
//	    ID    string `gorm:"primarykey"`
//	    Name  string
//	    Email string
//	}
//
//	var orders []Order
//	err := Database[*Order]().
//	    WithSelectRaw("orders.*, users.name as user_name").
//	    WithJoinRaw("LEFT JOIN users ON users.id = orders.user_id").
//	    List(&orders)
//
// 2. Multi-table join query:
//
//	var details []OrderDetail
//	err := Database[*OrderDetail]().
//	    WithSelectRaw("order_details.*, orders.amount, products.name as product_name").
//	    WithJoinRaw("LEFT JOIN orders ON orders.id = order_details.order_id").
//	    WithJoinRaw("LEFT JOIN products ON products.id = order_details.product_id").
//	    List(&details)
//
// 3. Query orders with active users:
//
//	var orders []Order
//	err := Database[*Order]().
//	    WithSelectRaw("orders.*, users.name").
//	    WithJoinRaw("LEFT JOIN users ON users.id = orders.user_id AND users.active = ?", 1).
//	    List(&orders)
//
// 4. Complex query with multiple conditions:
//
//	var orders []Order
//	err := Database[*Order]().
//	    WithSelectRaw("orders.*, users.name, products.name as product_name").
//	    WithJoinRaw("LEFT JOIN users ON users.id = orders.user_id").
//	    WithJoinRaw("LEFT JOIN order_details ON order_details.order_id = orders.id").
//	    WithJoinRaw("LEFT JOIN products ON products.id = order_details.product_id").
//	    WithTimeRange("orders.created_at", startTime, endTime).
//	    WithOrder("orders.created_at DESC").
//	    WithScope(page, size).
//	    List(&orders)
func (db *database[M]) WithJoinRaw(query string, args ...any) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()

	query = strings.TrimSpace(query)
	if len(query) == 0 {
		return db
	}

	upperQuery := strings.ToUpper(query)
	if !strings.Contains(upperQuery, "JOIN") || !strings.Contains(upperQuery, "ON") {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("WithJoinRaw")).Warnz("invalid join clause, must contain JOIN and ON",
			zap.String("query", query),
			zap.String("table", db.typ.Name()),
		)
		return db
	}

	db.ins = db.ins.Joins(query, args...)
	return db
}

// WithGroup adds GROUP BY clause to the query for data aggregation.
// Used with aggregate functions like COUNT, SUM, AVG, etc.
//
// Parameters:
//   - name: Column name or expression to group by
//
// Example:
//
//	WithGroup("category_id")  // Group by category
//	WithGroup("DATE(created_at)")  // Group by date
//
// WithGroup adds GROUP BY clause to SELECT statement.
// For example:
//
//	// Basic group by
//	db.WithGroup("user_id")
//
//	// Group by multiple columns
//	db.WithGroup("user_id, order_status")
//
//	// Common usage with aggregate functions
//	db.WithSelectRaw("user_id, COUNT(*) as order_count, SUM(amount) as total_amount").
//	   WithGroup("user_id")
//
// Note: WithGroup is typically used with aggregate functions (COUNT, SUM, AVG, etc.)
// and should be combined with WithSelectRaw to specify the grouped fields.
func (db *database[M]) WithGroup(name string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	name = strings.TrimSpace(name)
	if len(name) > 0 {
		db.ins = db.ins.Group(name)
	}
	return db
}

// WithHaving adds HAVING clause to the query for filtering grouped results.
// Used in conjunction with GROUP BY to filter aggregated data.
//
// Parameters:
//   - query: HAVING condition expression
//   - args: Optional arguments for parameterized conditions
//
// Example:
//
//	WithGroup("category_id").WithHaving("COUNT(*) > ?", 5)
//	WithHaving("SUM(amount) > 1000")
//
// WithHaving adds HAVING clause to filter grouped records.
// HAVING clause is used to filter groups, similar to WHERE but operates on grouped records.
// For example:
//
//	// Basic having clause
//	db.WithHaving("COUNT(*) > ?", 5)
//
//	// With aggregate functions
//	db.WithSelectRaw("user_id, COUNT(*) as order_count, SUM(amount) as total_amount").
//	   WithGroup("user_id").
//	   WithHaving("SUM(amount) > ?", 1000)
//
//	// Multiple conditions
//	db.WithHaving("COUNT(*) > ? AND SUM(amount) > ?", 5, 1000)
//
// Note: WithHaving must be used with GROUP BY clause and aggregate functions.
// WithHaving adds HAVING clause to filter grouped results.
// Used with GROUP BY to apply conditions on aggregated data.
// Similar to WHERE but operates on grouped results after aggregation.
//
// Parameters:
//   - query: HAVING condition (string, map, or struct)
//   - args: Optional arguments for parameterized queries
//
// Examples:
//
//	WithHaving("COUNT(*) > ?", 5)                    // Groups with more than 5 records
//	WithHaving("AVG(price) > 100")                   // Groups with average price > 100
//	WithHaving("SUM(amount) BETWEEN ? AND ?", 1000, 5000)  // Sum within range
//	WithHaving(map[string]any{"COUNT(*)": 10})       // Exact count match
//
// Note: HAVING clause is typically used after GROUP BY operations.
// Use WithGroup() before WithHaving() for proper SQL structure.
func (db *database[M]) WithHaving(query any, args ...any) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.ins = db.ins.Having(query, args...)
	return db
}

// WithOrder adds ORDER BY clause to sort query results.
// Supports multiple sorting criteria and directions (ASC/DESC).
// Column names are automatically wrapped with backticks to handle SQL keywords.
//
// Parameters:
//   - order: Column name(s) with optional direction. Multiple columns separated by commas.
//     Direction can be "ASC" (default) or "DESC" (case-insensitive).
//
// Examples:
//
//	WithOrder("name")                        // Sort by name ascending (default)
//	WithOrder("name ASC")                    // Sort by name ascending (explicit)
//	WithOrder("name asc")                    // Sort by name ascending (case-insensitive)
//	WithOrder("created_at DESC")             // Sort by creation date descending
//	WithOrder("created_at desc")             // Sort by creation date descending (case-insensitive)
//	WithOrder("priority DESC, name ASC")     // Multiple sort criteria
//	WithOrder("priority desc, name asc")     // Multiple sort criteria (case-insensitive)
//	WithOrder("order DESC, limit ASC")       // Handles SQL keywords safely
//
// Note:
//   - Column names are automatically escaped with backticks to prevent SQL injection
//     and handle reserved keywords like "order", "limit", etc.
//   - Direction keywords (ASC/DESC) are case-insensitive and will be converted to uppercase.
func (db *database[M]) WithOrder(order string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	// 可以多多个字段进行排序, 每个字段之间通过逗号分隔,
	// order 的值比如: "field1, field2 desc, field3 asc"
	items := strings.SplitSeq(order, ",")
	for _order := range items {
		if len(order) != 0 {
			items := strings.Fields(_order)
			for i := range items {
				if strings.EqualFold(items[i], "asc") || strings.EqualFold(items[i], "desc") {
					items[i] = strings.ToUpper(items[i])
				} else {
					// 第一个是排序字段,必须加上反引号,因为排序的字符串可能是 sql 语句关键字
					// 第二个可能是 "desc" 等关键字不需要加反引号
					// items[0] = "`" + items[0] + "`"
					// 如果不是关键字都加上反引号
					items[i] = "`" + items[i] + "`"
				}
			}
			_orders := strings.Join(items, " ")
			db.ins = db.ins.Order(_orders)
		}
	}
	return db
}

// WithPagination applies pagination parameters to the query.
// It calculates the offset based on the page and size parameters and applies
// the OFFSET and LIMIT clauses to the query.
//
// Parameters:
//   - page: The page number (1-based). If page <= 0, it defaults to 1.
//   - size: The number of records per page. If size <= 0, it defaults to defaultLimit.
//
// Examples:
//   - pageStr, _ := c.GetQuery("page")
//     sizeStr, _ := c.GetQuery("size")
//     page, _ := strconv.Atoi(pageStr)
//     size, _ := strconv.Atoi(sizeStr)
//     WithPagination(page, size)
func (db *database[M]) WithPagination(page, size int) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultLimit
	}
	offset := (page - 1) * size
	db.ins = db.ins.Scopes(func(d *gorm.DB) *gorm.DB {
		return d.Offset(offset).Limit(size)
	})
	return db
}

// WithLimit adds LIMIT clause to restrict the number of returned records.
// Used for pagination and controlling result set size.
//
// Parameters:
//   - limit: Maximum number of records to return (must be positive)
//
// Returns the same instance if limit is not positive.
//
// Example:
//
//	WithLimit(10)  // Return at most 10 records
//	WithLimit(100).WithOffset(20)  // Pagination: skip 20, take 100
//
// WithLimit specifies the number of record to be retrieved.
func (db *database[M]) WithLimit(limit int) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.ins = db.ins.Limit(limit)
	return db
}

// WithExpand enables eager loading of specified associations.
// Preloads related data to avoid N+1 query problems.
//
// Parameters:
//   - query: Association name or nested association path
//   - args: Optional conditions for the preloaded association
//
// Example:
//
//	WithExpand("Orders")  // Preload user's orders
//	WithExpand("Orders.Items")  // Preload nested associations
//	WithExpand("Orders", "status = ?", "completed")  // Conditional preload
//
// WithExpand preload associations with given conditions.
// order: preload with order.
// eg: [Children.Children.Children Parent.Parent.Parent]
// eg: [Children Parent]
//
// NOTE: WithExpand only workds on mysql foreign key relationship.
// If you want expand the custom field that without gorm tag about foreign key definition,
// you should define the GetAfter/ListAfter in model layer or service layoer.
// WithExpand enables eager loading of related models with optional ordering.
// It uses GORM's Preload functionality to load associated data in a single query.
//
// Parameters:
//   - expand: Slice of relationship names to preload (e.g., ["Children", "Parent"])
//   - order: Optional ordering for the preloaded relationships
//
// Example:
//
//	// Load user with their posts
//	db.WithExpand([]string{"Posts"})
//
//	// Load user with posts ordered by creation date
//	db.WithExpand([]string{"Posts"}, "created_at desc")
//
//	// Load nested relationships
//	db.WithExpand([]string{"Posts.Comments", "Profile"})
//
// Note: The first field in the order string will be wrapped with backticks
// to handle SQL keywords properly.
func (db *database[M]) WithExpand(expand []string, order ...string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	var _orders string
	if len(order) > 0 {
		if len(order[0]) > 0 {
			items := strings.Fields(order[0])
			// The first item is the sort field, must be wrapped with backticks
			// because the sort string might be a SQL keyword
			// The second item might be "desc" etc., which doesn't need backticks
			items[0] = "`" + items[0] + "`"
			_orders = strings.Join(items, " ")
		}
	}
	withOrder := func(db *gorm.DB) *gorm.DB {
		if len(_orders) > 0 {
			return db.Order(_orders)
		} else {
			return db
		}
	}
	// FIXME: 前端加了 _depth 查询参数, 但是层数不匹配就无法递归排序,
	// _depth 的作用:
	// _depth = 2: Children -> Children.Children
	// _depth = 3: Children -> Children.Children.Children
	// 假设一共有3层, 但是 _depth=5, 则无法递归排序
	//
	// 解决办法:
	// 假设: [Children.Children.Children, Parent]
	// 以前:
	//      db.db = db.db.Preload("Children.Children.Children", withOrder)
	//      db.db = db.db.Preload("Parent", withOrder)
	// 现在: (递归 Children)
	//      db.db = db.db.Preload("Children", withOrder)
	//      db.db = db.db.Preload("Children.Children", withOrder)
	//      db.db = db.db.Preload("Children.Children.Children", withOrder)
	//      db.db = db.db.Preload("Parent", withOrder)

	for i := range expand {
		// preload 排序问题
		// https://www.jianshu.com/p/a88fb2d4b2ef
		// https://gorm.io/docs/preload.html#Custom-Preloading-SQL

		items := strings.Split(expand[i], ".")
		switch len(items) {
		case 0:
		case 1:
			db.ins = db.ins.Preload(expand[i], withOrder)
		default:
			for j := range items {
				// fmt.Println("================== ", strings.Join(items[0:j+1], "."))
				db.ins = db.ins.Preload(strings.Join(items[0:j+1], "."), withOrder)
			}
		}
	}

	return db
}

// WithExclude excludes records that match specified conditions.
// It adds NOT conditions to the query to filter out records with matching values.
//
// Parameters:
//   - excludes: Map where keys are field names and values are slices of values to exclude
//
// Example:
//
//	// Exclude users with specific IDs
//	excludes := map[string][]any{
//		"id": {"user1", "user2", "user3"},
//	}
//	db.WithExclude(excludes)
//
//	// Exclude users with specific IDs and names
//	excludes := map[string][]any{
//		"id":   {"user1", "user2"},
//		"name": {"admin", "root"},
//	}
//	db.WithExclude(excludes)
//
// Note: This method affects the WHERE clause, not the SELECT clause.
// Use WithOmit() to exclude fields from SELECT queries.
func (db *database[M]) WithExclude(excludes map[string][]any) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	for k, v := range excludes {
		db.ins = db.ins.Not(k, v)
	}
	return db
}

// WithPurge explicitly controls whether to permanently delete records (hard delete).
// This option has the HIGHEST PRIORITY and overrides the model's default Purge() behavior.
//
// Priority order:
//  1. WithPurge() - explicitly set by user (highest priority)
//  2. model.Purge() - default behavior defined in the model
//  3. false - framework default (soft delete)
//
// Parameters:
//   - enable: Optional boolean flag (default: true if omitted)
//   - true: Hard delete (permanent deletion, bypasses soft delete)
//   - false: Soft delete (only updates deleted_at field)
//
// Usage:
//
//	WithPurge().Delete(&user)        // Hard delete (enable=true by default)
//	WithPurge(true).Delete(&user)    // Hard delete (explicit)
//	WithPurge(false).Delete(&user)   // Soft delete (explicit, overrides model.Purge())
//
// WARNING: Hard delete will permanently remove data from the database and cannot be undone.
// Only works on 'Delete' method.
func (db *database[M]) WithPurge(enable ...bool) types.Database[M] {
	_enable := true
	if len(enable) > 0 {
		_enable = enable[0]
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	db.enablePurge = util.ValueOf(_enable)
	return db
}

// WithCache enables query result caching with specified TTL (Time To Live).
// Improves performance by storing frequently accessed data in memory.
//
// Parameters:
//   - ttl: Cache duration (time.Duration)
//
// Example:
//
//	WithCache().List(&users)
//
// WithCache will make query resource count from cache.
// If cache not found or expired. query from database directly.
func (db *database[M]) WithCache(enable ...bool) types.Database[M] {
	_enable := true
	if len(enable) > 0 {
		_enable = enable[0]
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	db.enableCache = _enable
	return db
}

// WithOmit excludes specified fields from INSERT and UPDATE operations.
// Useful for skipping auto-generated fields or fields that shouldn't be modified.
//
// Parameters:
//   - columns: Field names to omit from the operation
//
// Example:
//
//	WithOmit("created_at", "updated_at").Create(&user)  // Skip timestamp fields
//	WithOmit("id").Update(&user)  // Skip ID field during update
//
// WithOmit omit specific columns when create/update/query.
func (db *database[M]) WithOmit(columns ...string) types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.ins = db.ins.Omit(columns...)
	return db
}

// WithDryRun enables dry-run mode to preview SQL queries without executing them.
// Useful for debugging, query optimization, and testing query generation.
// The generated SQL will be logged but not executed against the database.
//
// Example:
//
//	WithDryRun().Create(&user)  // Preview INSERT SQL without creating record
//	WithDryRun().WithQuery(params).List(&users)  // Preview SELECT SQL
//
// WithDryRun only executes model hooks without performing actual database operations.
// Also logs the SQL statements that would have been executed.
func (db *database[M]) WithDryRun() types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.dryRun = true
	return db
}

// WithoutHook disables model hooks (callbacks) for the current operation.
// Bypasses BeforeCreate, AfterCreate, BeforeUpdate, AfterUpdate, etc. hooks.
// Use when you need direct database operations without business logic interference.
//
// Example:
//
//	WithoutHook().Create(&user)  // Create without triggering hooks
//	WithoutHook().Update(&user)  // Update without validation hooks
//
// WithoutHook will disable all model hooks.
func (db *database[M]) WithoutHook() types.Database[M] {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.noHook = true
	return db
}

// Create inserts one or multiple records into the database.
// Automatically sets ID (if empty), created_at, and updated_at timestamps.
// Executes CreateBefore and CreateAfter model hooks unless disabled with WithoutHook.
// Supports batch processing for large datasets using configurable batch sizes.
//
// Parameters:
//   - objs: One or more model instances to create
//
// Returns error if validation fails, database constraints are violated, or hooks return errors.
//
// Example:
//
//	Create(&User{Name: "John", Email: "john@example.com"})
//	Create(user1, user2, user3)  // Batch create multiple records
func (db *database[M]) Create(_objs ...M) (err error) {
	if len(_objs) == 0 {
		return nil
	}
	var empty M
	objs := make([]M, 0, len(_objs))
	for _, obj := range _objs {
		if reflect.DeepEqual(obj, empty) {
			continue
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}

	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("Create", len(objs))
	defer done(err)
	if len(objs) == 0 {
		return nil
	}

	if db.enableCache {
		defer cache.Cache[[]M]().WithContext(ctx).Clear()
	}
	// if config.App.RedisConfig.Enable {
	// 	defer func() {
	// 		go func() {
	// 			begin := time.Now()
	// 			prefix, _ := buildCacheKey(db.db.Model(*new(M)).Session(&gorm.Session{DryRun: true}).Statement, "create")
	// 			defer logger.Cache.Infow("remove cache after create", "cost", time.Since(begin).String(), "prefix", prefix)
	// 			if err = redis.RemovePrefix(prefix); err != nil {
	// 				logger.Cache.Errorw("failed to remove cache keys", err, "action", "create")
	// 			}
	// 		}()
	// 	}()
	// }

	// Invoke model hook: CreateBefore for the entire batch.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_CREATE_BEFORE, span, func(spanCtx context.Context) error {
			for i := range objs {
				if !reflect.DeepEqual(empty, objs[i]) {
					if err = objs[i].CreateBefore(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	for i := range objs {
		if !reflect.DeepEqual(empty, objs[i]) {
			objs[i].SetID() // set id when id is empty.
		}
	}

	// if err = db.db.Save(objs).Error; err != nil {
	// if err = db.db.Table(db.tableName).Save(objs).Error; err != nil {
	// 	return err
	// }
	//
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	batchSize := defaultBatchSize
	if db.batchSize > 0 {
		batchSize = db.batchSize
	}
	// update "created_at" and "updated_at"
	now := time.Now()
	for i := range len(objs) {
		objs[i].SetCreatedAt(now)
		objs[i].SetUpdatedAt(now)
	}
	for i := 0; i < len(objs); i += batchSize {
		end := min(i+batchSize, len(objs))
		if err = db.ins.Session(&gorm.Session{DryRun: db.dryRun}).Table(tableName).Save(objs[i:end]).Error; err != nil {
			return err
		}
	}
	if db.enableCache {
		for i := range objs {
			_ = cache.Cache[M]().WithContext(ctx).Delete(objs[i].GetID())
		}
	}

	// // because db.db.Delete method just update field "delete_at" to current time,
	// // not really delete it(soft delete).
	// // If record already exists, Update method update all fields but exclude "created_at" by
	// // mysql "ON DUPLICATE KEY UPDATE" mechanism. so we should update the "created_at" field manually.
	// for i := range objs {
	// 	// 有些 model 重写 SetID 为一个空函数, 则 GetID() 的值为空字符串. 更新 created_at 则会报错
	// 	// 例如 casbin_rule 表/结构体: 这张表的 ID 总是 integer 类型, 并且有 autoincrement 属性, 所以必须重写 SetID.
	// 	if len(objs[i].GetID()) == 0 {
	// 		continue
	// 	}
	//
	// 	// 这里要重新创建一个 gorm.DB 实例, 否则会出现这种语句, id 出现多次了.
	// 	// UPDATE `assets` SET `created_at`='2023-11-12 14:35:42.604',`updated_at`='2023-11-12 14:35:42.604' WHERE id = '010103NU000020' AND `assets`.`deleted_at` IS NULL AND id = '010103NU000021' AND id = '010103NU000022' LIMIT 1000
	// 	var _db *gorm.DB
	// 	if strings.ToLower(config.App.Logger.Level) == "debug" {
	// 		_db = DB.Debug()
	// 	} else {
	// 		_db = DB
	// 	}
	// 	createdAt := time.Now()
	// 	if err = _db.Table(tableName).Model(*new(M)).Where("id = ?", objs[i].GetID()).Update("created_at", createdAt).Error; err != nil {
	// 		return err
	// 	}
	// }

	// Invoke model hook: CreateAfter for the entire batch.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_CREATE_AFTER, span, func(spanCtx context.Context) error {
			for i := range objs {
				if !reflect.DeepEqual(empty, objs[i]) {
					if err = objs[i].CreateAfter(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes one or multiple records from the database.
// By default performs soft delete (sets deleted_at timestamp).
// Use WithPurge() for permanent deletion (hard delete).
// Executes DeleteBefore and DeleteAfter model hooks unless disabled with WithoutHook.
//
// Parameters:
//   - objs: One or more model instances to delete
//
// Behavior:
//   - Soft delete: Sets deleted_at field, records remain in database
//   - Hard delete (with WithPurge): Permanently removes records
//   - Supports batch processing for performance
//
// Example:
//
//	Delete(&user)  // Soft delete by primary key
//	WithQuery(params).Delete(&User{})  // Delete with conditions
//	WithPurge().Delete(&user)  // Permanent deletion
func (db *database[M]) Delete(_objs ...M) (err error) {
	if len(_objs) == 0 {
		return nil
	}
	var empty M
	objs := make([]M, 0, len(_objs))
	for _, obj := range _objs {
		if reflect.DeepEqual(obj, empty) {
			continue
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}

	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("Delete", len(objs))
	defer done(err)
	if len(objs) == 0 {
		return nil
	}

	if db.enableCache {
		defer cache.Cache[[]M]().WithContext(ctx).Clear()
	}
	// if config.App.RedisConfig.Enable {
	// 	defer func() {
	// 		// TODO:only delete cache of all list statement and cache for current get statements.
	// 		go func() {
	// 			begin := time.Now()
	// 			prefix, _ := buildCacheKey(db.db.Model(*new(M)).Session(&gorm.Session{DryRun: true}).Statement, "delete")
	// 			defer logger.Cache.Infow("remove cache after delete", "cost", time.Since(begin).String(), "prefix", prefix)
	// 			if err = redis.RemovePrefix(prefix); err != nil {
	// 				logger.Cache.Errorw("failed to remove cache keys", err, "action", "delete")
	// 			}
	// 		}()
	// 	}()
	// }

	// Invoke model hook: DeleteBefore.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_DELETE_BEFORE, span, func(spanCtx context.Context) error {
			for i := range objs {
				if !reflect.DeepEqual(empty, objs[i]) {
					if err = objs[i].DeleteBefore(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	if util.Deref(db.enablePurge) {
		// delete permanently.
		// if err = db.db.Unscoped().Delete(objs).Error; err != nil {
		// if err = db.db.Table(db.tableName).Unscoped().Delete(objs).Error; err != nil {
		// 	return err
		// }
		//
		batchSize := defaultDeleteBatchSize
		if db.batchSize > 0 {
			batchSize = db.batchSize
		}
		for i := 0; i < len(objs); i += batchSize {
			end := min(i+batchSize, len(objs))
			if err = db.ins.Session(&gorm.Session{DryRun: db.dryRun}).Table(tableName).Unscoped().Delete(objs[i:end]).Error; err != nil {
				return err
			}
			if db.enableCache {
				_ = cache.Cache[M]().WithContext(ctx).Delete(objs[i].GetID())
			}
		}
	} else {
		// Delete() method just update field "delete_at" to currrent time.
		// DO NOT FORGET update the "created_at" field when create/update if record already exists.
		// if err = db.db.Delete(objs).Error; err != nil {
		// if err = db.db.Table(db.tableName).Delete(objs).Error; err != nil {
		// 	return err
		// }
		//
		batchSize := defaultDeleteBatchSize
		if db.batchSize > 0 {
			batchSize = db.batchSize
		}
		for i := 0; i < len(objs); i += batchSize {
			end := min(i+batchSize, len(objs))
			if err = db.ins.Session(&gorm.Session{DryRun: db.dryRun}).Table(tableName).Delete(objs[i:end]).Error; err != nil {
				return err
			}
			if db.enableCache {
				_ = cache.Cache[M]().WithContext(ctx).Delete(objs[i].GetID())
			}
		}
	}
	// Invoke model hook: DeleteAfter.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_DELETE_AFTER, span, func(spanCtx context.Context) error {
			for i := range objs {
				if !reflect.DeepEqual(empty, objs[i]) {
					if err = objs[i].DeleteAfter(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// Update modifies one or multiple records in the database.
// Automatically updates the updated_at timestamp for each record.
// Executes UpdateBefore and UpdateAfter model hooks unless disabled with WithoutHook.
// Uses GORM's Save method which performs INSERT or UPDATE based on primary key existence.
//
// Parameters:
//   - objs: One or more model instances to update
//
// Behavior:
//   - Sets ID if empty before updating
//   - Updates all fields of the model
//   - Supports batch processing for performance
//   - Clears related cache entries
//
// Example:
//
//	user.Name = "Updated Name"
//	Update(&user)  // Update single record
//	Update(user1, user2, user3)  // Batch update multiple records
func (db *database[M]) Update(_objs ...M) (err error) {
	if len(_objs) == 0 {
		return nil
	}
	var empty M
	objs := make([]M, 0, len(_objs))
	for _, obj := range _objs {
		if reflect.DeepEqual(obj, empty) {
			continue
		}
		objs = append(objs, obj)
	}
	if len(objs) == 0 {
		return nil
	}

	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("Update", len(objs))
	defer done(err)
	if len(objs) == 0 {
		return nil
	}

	if db.enableCache {
		defer cache.Cache[[]M]().WithContext(ctx).Clear()
	}
	// if config.App.RedisConfig.Enable {
	// 	defer func() {
	// 		go func() {
	// 			begin := time.Now()
	// 			prefix, _ := buildCacheKey(db.db.Model(*new(M)).Session(&gorm.Session{DryRun: true}).Statement, "update")
	// 			defer logger.Cache.Infow("remove cache after update", "cost", time.Since(begin).String(), "prefix", prefix)
	// 			if err = redis.RemovePrefix(prefix); err != nil {
	// 				logger.Cache.Errorw("failed to remove cache keys", err, "action", "update")
	// 			}
	// 		}()
	// 	}()
	// }

	// Invoke model hook: UpdateBefore.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_UPDATE_BEFORE, span, func(spanCtx context.Context) error {
			for i := range objs {
				if !reflect.DeepEqual(empty, objs[i]) {
					if err = objs[i].UpdateBefore(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	for i := range objs {
		if !reflect.DeepEqual(empty, objs[i]) {
			objs[i].SetID() // set id when id is empty
		}
	}
	// if err = db.db.Save(objs).Error; err != nil {
	// if err = db.db.Table(db.tableName).Save(objs).Error; err != nil {
	// 	return err
	// }
	//
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	batchSize := defaultBatchSize
	if db.batchSize > 0 {
		batchSize = db.batchSize
	}
	for i := 0; i < len(objs); i += batchSize {
		end := min(i+batchSize, len(objs))
		if err = db.ins.Session(&gorm.Session{DryRun: db.dryRun}).Table(tableName).Save(objs[i:end]).Error; err != nil {
			zap.S().Error(err)
			return err
		}
		if db.enableCache {
			for j := range objs[i:end] {
				_ = cache.Cache[M]().WithContext(ctx).Delete(objs[j].GetID())
			}
		}
	}
	// Invoke model hook: UpdateAfter.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_UPDATE_AFTER, span, func(spanCtx context.Context) error {
			for i := range objs {
				if !reflect.DeepEqual(empty, objs[i]) {
					if err = objs[i].UpdateAfter(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// UpdateByID updates a specific field of a single record identified by ID.
// This is a lightweight update operation that bypasses model hooks for performance.
// Only updates the specified field without triggering validation or business logic.
//
// Parameters:
//   - id: The primary key of the record to update
//   - name: The field name to update
//   - value: The new value for the field
//
// Note: Does not invoke UpdateBefore/UpdateAfter hooks for performance reasons.
//
// Example:
//
//	UpdateById("user123", "status", "active")  // Update user status
//	UpdateById("order456", "amount", 99.99)    // Update order amount
func (db *database[M]) UpdateByID(id string, name string, value any) (err error) {
	if len(id) == 0 {
		logger.Database.Warn("empty id")
		return nil
	}
	if len(name) == 0 {
		logger.Database.Warn("empty name")
		return nil
	}
	if value == nil {
		logger.Database.Warn("empty value")
		return nil
	}

	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, _ := db.trace("UpdateById")
	defer done(err)

	if db.enableCache {
		defer cache.Cache[[]M]().WithContext(ctx).Clear()
	}
	// if config.App.RedisConfig.Enable {
	// 	defer func() {
	// 		go func() {
	// 			begin := time.Now()
	// 			prefix, _ := buildCacheKey(db.db.Model(*new(M)).Session(&gorm.Session{DryRun: true}).Statement, "update_by_id")
	// 			defer logger.Cache.Infow("remove cache after update_by_id", "cost", time.Since(begin).String(), "prefix", prefix)
	// 			if err = redis.RemovePrefix(prefix); err != nil {
	// 				logger.Cache.Errorw("failed to remove cache keys", err, "action", "update")
	// 			}
	// 		}()
	// 	}()
	// }

	// return db.db.Model(*new(M)).Where("id = ?", id).Update(name, value).Error
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	if err = db.ins.Session(&gorm.Session{DryRun: db.dryRun}).Table(tableName).Model(*new(M)).Where("id = ?", id).Update(name, value).Error; err != nil {
		return err
	}
	if db.enableCache {
		_ = cache.Cache[M]().WithContext(ctx).Delete(id)
	}
	return nil
}

// List retrieves multiple records from the database based on applied conditions.
// Returns all records if no conditions are specified, or filtered records with WithQuery.
// Supports caching, pagination, sorting, and eager loading of associations.
//
// Parameters:
//   - dest: Pointer to slice where results will be stored
//   - _cache: Optional cache parameter for advanced caching control
//
// Features:
//   - Automatic result caching when enabled
//   - Supports pagination with WithLimit/WithOffset
//   - Supports sorting with WithOrder
//   - Supports filtering with WithQuery
//   - Supports eager loading with WithExpand
//
// Example:
//
//	var users []User
//	List(&users)  // Get all users
//	WithQuery(&User{Status: "active"}).List(&users)  // Get active users
//	WithLimit(10).WithOffset(20).List(&users)  // Paginated results
func (db *database[M]) List(dest *[]M, _cache ...*[]byte) (err error) {
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("List")
	defer done(err)
	if dest == nil {
		return nil
	}

	begin := time.Now()
	var key string
	if !db.enableCache {
		goto QUERY
	}
	_, _, key = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).Find(dest).Statement, "list")
	if _dest, e := cache.Cache[[]M]().WithContext(ctx).Get(key); e != nil {
		// metrics.CacheMiss.WithLabelValues("list", db.typ.Name()).Inc()
		goto QUERY
	} else {
		// metrics.CacheHit.WithLabelValues("list", db.typ.Name()).Inc()
		*dest = _dest
		logger.Cache.Infow("list from cache", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		return nil
	}

	// =============================
	// ===== BEGIN redis cache =====
	// =============================
	// begin := time.Now()
	// var key string
	// var shouldDecode bool // if cache is nil or cache[0] is nil, we should decod the queryed cache in to "dest".
	// var _cache []byte
	// if !db.enableCache {
	// 	goto QUERY
	// }
	// if !config.App.RedisConfig.Enable {
	// 	goto QUERY
	// }
	// _, key = buildCacheKey(db.db.Session(&gorm.Session{DryRun: true}).Find(dest).Statement, "list")
	// if len(cache) == 0 {
	// 	shouldDecode = true
	// } else {
	// 	if cache[0] == nil {
	// 		shouldDecode = true
	// 	}
	// }
	// if shouldDecode {
	// 	var _dest []M
	// 	if _dest, err = redis.GetML[M](key); err == nil {
	// 		val := reflect.ValueOf(dest)
	// 		if val.Kind() != reflect.Pointer || val.Elem().Kind() != reflect.Slice {
	// 			return ErrNotPtrSlice
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableSlice
	// 		}
	// 		if !val.Elem().CanSet() {
	// 			return ErrNotSetSlice
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_dest))
	// 		logger.Cache.Infow("list and decode from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// } else {
	// 	if _cache, err = redis.Get(key); err == nil {
	// 		val := reflect.ValueOf(cache[0])
	// 		if val.Kind() != reflect.Pointer || val.Elem().Kind() != reflect.Slice {
	// 			return ErrNotPtrSlice
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableSlice
	// 		}
	// 		if !val.Elem().CanSet() {
	// 			return ErrNotSetSlice
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_cache))
	// 		logger.Cache.Infow("list from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// }
	// if !errors.Is(err, redis.ErrKeyNotExists) {
	// 	logger.Cache.Error(err)
	// 	return err
	// }
	// // Not Found cache and continue.
	// ===========================
	// ===== END redis cache =====
	// ===========================

QUERY:
	var empty M // call nil value M will cause panic.
	// Invoke model hook: ListBefore.
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_LIST_BEFORE, span, func(spanCtx context.Context) error {
			for i := range *dest {
				if !reflect.DeepEqual(empty, (*dest)[i]) {
					if err = (*dest)[i].ListBefore(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	// if err = db.db.Find(dest).Error; err != nil {
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	// apply cursor-based pagination.
	db.applyCursorPagination()
	if err = db.ins.Table(tableName).Find(dest).Error; err != nil {
		return err
	}
	// If cursor-based pagination is enabled and this is a previous page query,
	// reverse the list to mantain the original sort order.
	if db.enableCursor && !db.cursorNext {
		slices.Reverse(*dest)
	}

	// Invoke model hook: ListAfter()
	if !db.noHook {
		if err = traceModelHook[M](db.ctx, consts.PHASE_LIST_AFTER, span, func(spanCtx context.Context) error {
			for i := range *dest {
				if !reflect.DeepEqual(empty, (*dest)[i]) {
					if err = (*dest)[i].ListAfter(types.NewModelContext(db.ctx, spanCtx)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	// Cache the result.
	// if db.enableCache && config.App.RedisConfig.Enable {
	// 	logger.Cache.Infow("list from database", "cost", time.Since(begin).String(), "key", key)
	// 	go func() {
	// 		if err = redis.SetML[M](key, *dest); err != nil {
	// 			logger.Cache.Error(err)
	// 		}
	// 	}()
	// }
	if db.enableCache {
		logger.Cache.Infow("list from database", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		_ = cache.Cache[[]M]().WithContext(ctx).Set(key, *dest, config.App.Cache.Expiration)
	}

	return nil
}

// // Find equal to WithQuery(condition).List()
// // More detail see `List` document.
// func (db *database[T]) Find(dest *[]T, query T) error {
// 	return db.db.Where(query).Find(dest).Error
// }

// Get retrieves a single record from the database by its primary key (ID).
// Supports automatic caching to improve performance for frequently accessed records.
// Executes GetBefore and GetAfter model hooks unless disabled with WithoutHook.
//
// Parameters:
//   - dest: Pointer to model instance where the result will be stored
//   - id: Primary key value of the record to retrieve
//   - _cache: Optional cache parameter for advanced caching control
//
// Returns ErrIDRequired if id is empty, or database errors if record not found.
//
// Features:
//   - Automatic result caching when enabled
//   - Cache-first lookup for improved performance
//   - Supports eager loading with WithExpand
//   - Supports field selection with WithSelect
//
// Example:
//
//	var user User
//	Get(&user, "user123")  // Get user by ID
//	WithExpand("Orders").Get(&user, "user123")  // Get user with orders
func (db *database[M]) Get(dest M, id string, _cache ...*[]byte) (err error) {
	if len(id) == 0 {
		return ErrIDRequired
	}
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("Get")
	defer done(err)

	begin := time.Now()
	var key string
	if !db.enableCache {
		goto QUERY
	}
	_, _, key = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).Where("id = ?", id).Find(dest).Statement, "get", id)
	if _dest, e := cache.Cache[M]().WithContext(ctx).Get(key); e != nil {
		// metrics.CacheMiss.WithLabelValues("get", db.typ.Name()).Inc()
		goto QUERY
	} else {
		// metrics.CacheHit.WithLabelValues("get", db.typ.Name()).Inc()
		val := reflect.ValueOf(dest)
		if val.Kind() != reflect.Pointer {
			return ErrNotPtrStruct
		}
		if !val.Elem().CanAddr() {
			return ErrNotAddressableModel
		}
		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
		logger.Cache.Infow("get from cache", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		return nil
	}

	// =============================
	// ===== BEGIN redis cache =====
	// =============================
	// begin := time.Now()
	// var key string
	// var shouldDecode bool // if cache is nil or cache[0] is nil, we should decod the queryed cache in to "dest".
	// if !db.enableCache {
	// 	goto QUERY
	// }
	// if !config.App.RedisConfig.Enable {
	// 	goto QUERY
	// }
	// // _, key = BuildKey(db.db.Session(&gorm.Session{DryRun: true}).Where("id = ?", id).Find(dest).Statement, "get")
	// // 我发现这个 id 的值怎么都无法填充到 sql 语句中, 所以直接用 id 作为 key 的一部分,而不是用 sql 语句
	// // 如果不用 id 作为 redis key, 那么多个 get 的语句相同则 key 相同,获取到的都是同一个缓存.
	// _, key = buildCacheKey(db.db.Session(&gorm.Session{DryRun: true}).Where("id = ?", id).Find(dest).Statement, "get", id)
	// if len(cache) == 0 {
	// 	shouldDecode = true
	// } else {
	// 	if cache[0] == nil {
	// 		shouldDecode = true
	// 	}
	// }
	// if shouldDecode { // query cache from redis and decoded into 'dest'.
	// 	var _dest M
	// 	if _dest, err = redis.GetM[M](key); err == nil {
	// 		val := reflect.ValueOf(dest)
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrStruct
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableModel
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
	// 		logger.Cache.Infow("get and decode from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// } else {
	// 	var _cache []byte
	// 	if _cache, err = redis.Get(key); err == nil {
	// 		val := reflect.ValueOf(cache[0])
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrSlice
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableSlice
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_cache))
	// 		logger.Cache.Infow("get from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// }
	// if err != redis.ErrKeyNotExists {
	// 	logger.Cache.Error(err)
	// 	return err
	// }
	// // Not Found cache, continue query database.
	// ===========================
	// ===== END redis cache =====
	// ===========================

QUERY:
	var empty M // call nil value M will cause panic.
	// Invoke model hook: GetBefore.
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_BEFORE, span, func(spanCtx context.Context) error {
			return dest.GetBefore(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// if err = db.db.Where("id = ?", id).Find(dest).Error; err != nil {
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	// NOTE: In GORM v2, if the primary key field (e.g. "ID") is already set
	// on the struct `dest`, calling db.Find(dest) will automatically build
	// a query with a "WHERE primary_key = ?" clause.
	// This behavior does NOT exist in older versions of GORM,
	// where db.Find(dest) without Where(...) would scan the whole table.
	// To be safe across versions, db.First(dest, id) is explicit.
	//
	// dest.SetID(id)
	// if err = db.db.Table(tableName).Find(dest).Error; err != nil {
	// 	return err
	// }
	if len(tableName) == 0 {
		_, tableName, _ = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).Where("id = ?", id).Find(dest).Statement, "get", id)
	}
	dest.ClearID()
	if err = db.ins.Table(tableName).Where(fmt.Sprintf("`%s`.`id` = ?", tableName), id).Find(dest).Error; err != nil {
		return err
	}
	// Invoke model hook: GetAfter.
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_AFTER, span, func(spanCtx context.Context) error {
			return dest.GetAfter(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// // Cache the result.
	// if db.enableCache && config.App.RedisConfig.Enable {
	// 	logger.Cache.Infow("get from database", "cost", time.Since(begin).String(), "key", key)
	// 	go func() {
	// 		if err = redis.SetM[M](key, dest); err != nil {
	// 			logger.Cache.Error(err)
	// 		}
	// 	}()
	// }
	if db.enableCache {
		logger.Cache.Infow("get from database", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		_ = cache.Cache[M]().WithContext(ctx).Set(key, dest, config.App.Cache.Expiration)
	}
	return nil
}

// Count returns the total number of records matching the current query conditions.
// Supports automatic caching to improve performance for frequently executed count queries.
// Applies all previously set query conditions (WHERE, JOIN, etc.) to the count operation.
//
// Parameters:
//   - count: Pointer to int64 where the result count will be stored
//
// Returns database errors if the query fails.
//
// Features:
//   - Automatic result caching when enabled
//   - Cache-first lookup for improved performance
//   - Respects all query modifiers (WHERE, JOIN, GROUP BY, etc.)
//   - Uses LIMIT(-1) to ensure accurate count with existing LIMIT clauses
//
// Example:
//
//	var total int64
//	WithQuery("status = ?", "active").Count(&total)  // Count active records
//	WithJoinRaw("LEFT JOIN orders ON users.id = orders.user_id").Count(&total)  // Count with JOIN
//
// Note: The underlying type must be pointer to struct, otherwise panic will occur.
func (db *database[M]) Count(count *int64) (err error) {
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, _ := db.trace("Count")
	defer done(err)

	begin := time.Now()
	var key string
	if !db.enableCache {
		goto QUERY
	}
	_, _, key = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).Model(*new(M)).Count(count).Statement, "count")
	if _cache, e := cache.Cache[int64]().WithContext(ctx).Get(key); e != nil {
		// metrics.CacheMiss.WithLabelValues("count", db.typ.Name()).Inc()
		goto QUERY
	} else {
		// metrics.CacheHit.WithLabelValues("count", db.typ.Name()).Inc()
		*count = _cache
		logger.Cache.Infow("count from cache", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		return err
	}

	// =============================
	// ===== BEGIN redis cache =====
	// =============================
	// begin := time.Now()
	// var key string
	// var _cache int64
	// if count == nil {
	// 	return nil
	// }
	// if !db.enableCache {
	// 	goto QUERY
	// }
	// if !config.App.RedisConfig.Enable {
	// 	goto QUERY
	// }
	// fmt.Println("---- buildCacheKey count")
	// _, key = buildCacheKey(db.db.Session(&gorm.Session{DryRun: true}).Model(*new(M)).Count(count).Statement, "count")
	// if _cache, err = redis.GetInt(key); err == nil {
	// 	*count = _cache
	// 	logger.Cache.Infow("count from cache", "cost", time.Since(begin).String(), "key", key)
	// 	return
	// }
	// if !errors.Is(err, redis.ErrKeyNotExists) {
	// 	logger.Cache.Error(err)
	// 	return err
	// }
	// // NOT FOUND cache, continue query.
	// ===========================
	// ===== END redis cache =====
	// ===========================

QUERY:
	// if err = db.db.Model(*new(M)).Count(count).Error; err != nil {
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	if err = db.ins.Table(tableName).Model(*new(M)).Limit(-1).Count(count).Error; err != nil {
		logger.Cache.Error(err)
		return err
	}
	// if db.enableCache && config.App.RedisConfig.Enable {
	// 	logger.Cache.Infow("count from database", "cost", time.Since(begin).String(), "key", key)
	// 	go func() {
	// 		if err = redis.Set(key, *count); err != nil {
	// 			logger.Cache.Error(err)
	// 		}
	// 	}()
	//
	// }
	if db.enableCache {
		logger.Cache.Infow("count from database", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		_ = cache.Cache[int64]().WithContext(ctx).Set(key, *count, config.App.Cache.Expiration)

	}
	return nil
}

// First retrieves the first record from the database ordered by primary key.
// Supports automatic caching to improve performance for frequently accessed queries.
// Applies all previously set query conditions and returns the first matching record.
//
// Parameters:
//   - dest: Pointer to model instance where the result will be stored
//   - _cache: Optional cache parameter for advanced caching control
//
// Returns database errors if no record is found or query fails.
//
// Features:
//   - Automatic result caching when enabled
//   - Cache-first lookup for improved performance
//   - Supports all query modifiers (WHERE, ORDER BY, etc.)
//   - Supports eager loading with WithExpand
//   - Supports field selection with WithSelect
//
// Example:
//
//	var user User
//	First(&user)  // Get first user by primary key
//	WithQuery("status = ?", "active").First(&user)  // Get first active user
//	WithOrder("created_at DESC").First(&user)  // Get newest user
func (db *database[M]) First(dest M, _cache ...*[]byte) (err error) {
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("First")
	defer done(err)

	begin := time.Now()
	var key string
	if !db.enableCache {
		goto QUERY
	}
	_, _, key = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).First(dest).Statement, "first")
	if _dest, e := cache.Cache[M]().WithContext(ctx).Get(key); e != nil {
		// metrics.CacheMiss.WithLabelValues("first", db.typ.Name()).Inc()
		goto QUERY
	} else {
		// metrics.CacheHit.WithLabelValues("first", db.typ.Name()).Inc()
		val := reflect.ValueOf(dest)
		if val.Kind() != reflect.Pointer {
			return ErrNotPtrStruct
		}
		if !val.Elem().CanAddr() {
			return ErrNotAddressableModel
		}
		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
		logger.Cache.Infow("first from cache", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		return nil // Found cache and return.
	}

	// =============================
	// ===== BEGIN redis cache =====
	// =============================
	// begin := time.Now()
	// var key string
	// var shouldDecode bool // if cache is nil or cache[0] is nil, we should decod the queryed cache in to "dest".
	// if !db.enableCache {
	// 	goto QUERY
	// }
	// if !config.App.RedisConfig.Enable {
	// 	goto QUERY
	// }
	// _, key = buildCacheKey(db.db.Session(&gorm.Session{DryRun: true}).First(dest).Statement, "first")
	// if len(cache) == 0 {
	// 	shouldDecode = true
	// } else {
	// 	if cache[0] == nil {
	// 		shouldDecode = true
	// 	}
	// }
	// if shouldDecode { // query cache from redis and decode into 'dest'.
	// 	var _dest M
	// 	if _dest, err = redis.GetM[M](key); err == nil {
	// 		val := reflect.ValueOf(dest)
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrStruct
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableModel
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
	// 		logger.Cache.Infow("first and decode from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// } else {
	// 	var _cache []byte
	// 	if _cache, err = redis.Get(key); err == nil {
	// 		val := reflect.ValueOf(cache[0])
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrSlice
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableSlice
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_cache))
	// 		logger.Cache.Infow("first from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// 	if err != redis.ErrKeyNotExists {
	// 		logger.Cache.Error(err)
	// 		return err
	// 	}
	// }
	// Not Found cache, continue query database.
	// ===========================
	// ===== END redis cache =====
	// ===========================

QUERY:
	var empty M // call nil value M will cause panic.
	// Invoke model hook: GetBefore
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_BEFORE, span, func(spanCtx context.Context) error {
			return dest.GetBefore(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// if err = db.db.First(dest).Error; err != nil {
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	if err = db.ins.Table(tableName).First(dest).Error; err != nil {
		return err
	}
	// Invoke model hook: GetAfter
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_AFTER, span, func(spanCtx context.Context) error {
			return dest.GetAfter(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// // Cache the result.
	// if db.enableCache && config.App.RedisConfig.Enable {
	// 	logger.Cache.Infow("first from database", "cost", time.Since(begin).String(), "key", key)
	// 	go func() {
	// 		if err = redis.SetM[M](key, dest); err != nil {
	// 			logger.Cache.Error(err)
	// 		}
	// 	}()
	// }
	if db.enableCache {
		logger.Cache.Infow("first from database", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		_ = cache.Cache[M]().WithContext(ctx).Set(key, dest, config.App.Cache.Expiration)
	}
	return nil
}

// Last retrieves the last record from the database ordered by primary key.
// Supports automatic caching to improve performance for frequently accessed queries.
// Applies all previously set query conditions and returns the last matching record.
//
// Parameters:
//   - dest: Pointer to model instance where the result will be stored
//   - _cache: Optional cache parameter for advanced caching control
//
// Returns database errors if no record is found or query fails.
//
// Features:
//   - Automatic result caching when enabled
//   - Cache-first lookup for improved performance
//   - Supports all query modifiers (WHERE, ORDER BY, etc.)
//   - Supports eager loading with WithExpand
//   - Supports field selection with WithSelect
//   - Executes GetBefore and GetAfter model hooks unless disabled
//
// Example:
//
//	var user User
//	Last(&user)  // Get last user by primary key
//	WithQuery("status = ?", "active").Last(&user)  // Get last active user
//	WithOrder("created_at ASC").Last(&user)  // Get oldest user (with custom order)
func (db *database[M]) Last(dest M, _cache ...*[]byte) (err error) {
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("Last")
	defer done(err)

	begin := time.Now()
	var key string
	if !db.enableCache {
		goto QUERY
	}
	_, _, key = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).First(dest).Statement, "last")
	if _dest, e := cache.Cache[M]().WithContext(ctx).Get(key); e != nil {
		// metrics.CacheMiss.WithLabelValues("last", db.typ.Name()).Inc()
		goto QUERY
	} else {
		// metrics.CacheHit.WithLabelValues("last", db.typ.Name()).Inc()
		val := reflect.ValueOf(dest)
		if val.Kind() != reflect.Pointer {
			return ErrNotPtrStruct
		}
		if !val.Elem().CanAddr() {
			return ErrNotAddressableModel
		}
		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
		logger.Cache.Infow("last from cache", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		return nil // Found cache and return.
	}

	// =============================
	// ===== BEGIN redis cache =====
	// =============================
	// begin := time.Now()
	// var key string
	// var shouldDecode bool // if cache is nil or cache[0] is nil, we should decod the queryed cache in to "dest".
	// if !db.enableCache {
	// 	goto QUERY
	// }
	// if !config.App.RedisConfig.Enable {
	// 	goto QUERY
	// }
	// _, key = buildCacheKey(db.db.Session(&gorm.Session{DryRun: true}).First(dest).Statement, "last")
	// if len(cache) == 0 {
	// 	shouldDecode = true
	// } else {
	// 	if cache[0] == nil {
	// 		shouldDecode = true
	// 	}
	// }
	// if shouldDecode { // query cache from redis and decode into 'dest'.
	// 	var _dest M
	// 	if _dest, err = redis.GetM[M](key); err == nil {
	// 		val := reflect.ValueOf(dest)
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrStruct
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableModel
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
	// 		logger.Cache.Infow("last and decode from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// } else {
	// 	var _cache []byte
	// 	if _cache, err = redis.Get(key); err == nil {
	// 		val := reflect.ValueOf(cache[0])
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrSlice
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableSlice
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_cache))
	// 		logger.Cache.Infow("last from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// }
	// if err != redis.ErrKeyNotExists {
	// 	logger.Cache.Error(err)
	// 	return err
	// }
	// // Not Found cache, continue query database.
	// ===========================
	// ===== END redis cache =====
	// ===========================

QUERY:
	var empty M // call nil value M will cause panic.
	// Invoke model hook: GetBefore.
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_BEFORE, span, func(spanCtx context.Context) error {
			return dest.GetBefore(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// if err = db.db.Last(dest).Error; err != nil {
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	if err = db.ins.Table(tableName).Last(dest).Error; err != nil {
		return err
	}
	// Invoke model hook: GetAfter
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_AFTER, span, func(spanCtx context.Context) error {
			return dest.GetAfter(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// // Cache the result.
	// if db.enableCache && config.App.RedisConfig.Enable {
	// 	logger.Cache.Infow("last from database", "cost", time.Since(begin).String(), "key", key)
	// 	go func() {
	// 		if err = redis.SetM[M](key, dest); err != nil {
	// 			logger.Cache.Error(err)
	// 		}
	// 	}()
	// }
	if db.enableCache {
		logger.Cache.Infow("last from database", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		_ = cache.Cache[M]().WithContext(ctx).Set(key, dest, config.App.Cache.Expiration)
	}
	return nil
}

// Take retrieves the first record from the database in no specified order.
// Unlike First/Last which order by primary key, Take returns any matching record.
// Supports automatic caching to improve performance for frequently accessed queries.
//
// Parameters:
//   - dest: Pointer to model instance where the result will be stored
//   - _cache: Optional cache parameter for advanced caching control
//
// Returns database errors if no record is found or query fails.
//
// Features:
//   - Automatic result caching when enabled
//   - Cache-first lookup for improved performance
//   - Supports all query modifiers (WHERE, JOIN, etc.)
//   - Supports eager loading with WithExpand
//   - Supports field selection with WithSelect
//   - Executes GetBefore and GetAfter model hooks unless disabled
//   - No ordering applied (fastest single record retrieval)
//
// Example:
//
//	var user User
//	Take(&user)  // Get any user record
//	WithQuery("status = ?", "active").Take(&user)  // Get any active user
func (db *database[M]) Take(dest M, _cache ...*[]byte) (err error) {
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, ctx, span := db.trace("Take")
	defer done(err)

	begin := time.Now()
	var key string
	if !db.enableCache {
		goto QUERY
	}
	_, _, key = buildCacheKey(db.ins.Session(&gorm.Session{DryRun: true, Logger: glogger.Default.LogMode(glogger.Silent)}).First(dest).Statement, "take")
	if _dest, e := cache.Cache[M]().WithContext(ctx).Get(key); e != nil {
		// metrics.CacheMiss.WithLabelValues("take", db.typ.Name()).Inc()
		goto QUERY
	} else {
		// metrics.CacheHit.WithLabelValues("take", db.typ.Name()).Inc()
		val := reflect.ValueOf(dest)
		if val.Kind() != reflect.Pointer {
			return ErrNotPtrStruct
		}
		if !val.Elem().CanAddr() {
			return ErrNotAddressableModel
		}
		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
		logger.Cache.Infow("take from cache", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		return nil // Found cache and return.
	}

	// =============================
	// ===== BEGIN redis cache =====
	// =============================
	// begin := time.Now()
	// var key string
	// var shouldDecode bool // if cache is nil or cache[0] is nil, we should decod the queryed cache in to "dest".
	// if !db.enableCache {
	// 	goto QUERY
	// }
	// if !config.App.RedisConfig.Enable {
	// 	goto QUERY
	// }
	// _, key = buildCacheKey(db.db.Session(&gorm.Session{DryRun: true}).First(dest).Statement, "take")
	// if len(cache) == 0 {
	// 	shouldDecode = true
	// } else {
	// 	if cache[0] == nil {
	// 		shouldDecode = true
	// 	}
	// }
	// if shouldDecode { // query cache from redis and decode into 'dest'.
	// 	var _dest M
	// 	if _dest, err = redis.GetM[M](key); err == nil {
	// 		val := reflect.ValueOf(dest)
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrStruct
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableModel
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_dest).Elem()) // the type of M is pointer to struct.
	// 		logger.Cache.Infow("get and decode from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// } else {
	// 	var _cache []byte
	// 	if _cache, err = redis.Get(key); err == nil {
	// 		val := reflect.ValueOf(cache[0])
	// 		if val.Kind() != reflect.Pointer {
	// 			return ErrNotPtrSlice
	// 		}
	// 		if !val.Elem().CanAddr() {
	// 			return ErrNotAddressableSlice
	// 		}
	// 		val.Elem().Set(reflect.ValueOf(_cache))
	// 		logger.Cache.Infow("take from cache", "cost", time.Since(begin).String(), "key", key)
	// 		return nil // Found cache and return.
	// 	}
	// }
	// if err != redis.ErrKeyNotExists {
	// 	logger.Cache.Error(err)
	// 	return err
	// }
	// // Not Found cache, continue query database.
	// ===========================
	// ===== END redis cache =====
	// ===========================

QUERY:
	var empty M // call nil value M will cause panic.
	// Invoke model hook: GetBefore.
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_BEFORE, span, func(spanCtx context.Context) error {
			return dest.GetBefore(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// if err = db.db.Take(dest).Error; err != nil {
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	if err = db.ins.Table(tableName).Take(dest).Error; err != nil {
		return err
	}
	// Invoke model hook: GetAfter.
	if !db.noHook && !reflect.DeepEqual(empty, dest) {
		if err = traceModelHook[M](db.ctx, consts.PHASE_GET_AFTER, span, func(spanCtx context.Context) error {
			return dest.GetAfter(types.NewModelContext(db.ctx, spanCtx))
		}); err != nil {
			return err
		}
	}
	// // Cache the result.
	// if db.enableCache && config.App.RedisConfig.Enable {
	// 	logger.Cache.Infow("take from database", "cost", time.Since(begin).String(), "key", key)
	// 	go func() {
	// 		if err = redis.SetM[M](key, dest); err != nil {
	// 			logger.Cache.Error(err)
	// 		}
	// 	}()

	//
	// }
	if db.enableCache {
		logger.Cache.Infow("take from database", "cost", util.FormatDurationSmart(time.Since(begin)), "key", key)
		_ = cache.Cache[M]().WithContext(ctx).Set(key, dest, config.App.Cache.Expiration)
	}
	return nil
}

// Cleanup permanently deletes all soft-deleted records from the database.
// This operation removes records where 'deleted_at' column is not null.
// WARNING: This is a destructive operation that cannot be undone.
//
// Returns database errors if the cleanup operation fails.
//
// Features:
//   - Permanently removes soft-deleted records
//   - Uses unscoped delete to bypass soft delete protection
//   - Applies to all records in the table (ignores query conditions)
//   - Helps maintain database performance by removing obsolete data
//
// Example:
//
//	Cleanup()  // Remove all soft-deleted records
//
// Note: This operation affects the entire table and ignores any previously
// set query conditions. Use with caution in production environments.
func (db *database[M]) Cleanup() (err error) {
	if err = db.prepare(); err != nil {
		return err
	}
	defer db.reset()
	done, _, _ := db.trace("Cleanup")
	defer done(err)

	// return db.db.Limit(-1).Where("deleted_at IS NOT NULL").Model(*new(M)).Unscoped().Delete(make([]M, 0)).Error
	tableName := db.m.GetTableName() //nolint:errcheck
	if len(db.tableName) > 0 {
		tableName = db.tableName
	}
	return db.ins.Session(&gorm.Session{DryRun: db.dryRun}).Table(tableName).Limit(-1).Where("deleted_at IS NOT NULL").Model(*new(M)).Unscoped().Delete(make([]M, 0)).Error
}

// Health performs comprehensive database health checks including connectivity,
// connection pool status, and response time validation.
// Returns nil if all checks pass, otherwise returns detailed error information.
//
// Health checks performed:
//  1. Database connection test with SELECT 1 query
//  2. Connection pool status and capacity validation
//  3. Database ping test for response time measurement
//
// Returns database errors if any health check fails.
//
// Features:
//   - Comprehensive connectivity validation
//   - Connection pool monitoring and warnings
//   - Response time measurement
//   - Detailed logging of health status and metrics
//
// Example:
//
//	if err := Database[User]().Health(); err != nil {
//	  log.Fatal("Database unhealthy:", err)
//	}
func (db *database[M]) Health() error {
	if err := db.prepare(); err != nil {
		return err
	}
	defer db.reset()

	begin := time.Now()

	// 1.check database connection
	if err := db.ins.Exec("SELECT 1").Error; err != nil {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("Health")).Errorz("database connection check failed",
			zap.Error(err),
			zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
		)
		return fmt.Errorf("database connection check failed: %w", err)
	}

	// 2.check database connection pool
	sqlDB, err := db.ins.DB()
	if err != nil {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("Health")).Errorz("get sql.DB instance failed",
			zap.Error(err),
			zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
		)
		return fmt.Errorf("get sql.DB instance failed: %w", err)
	}

	// check database connection pool config
	stats := sqlDB.Stats()
	if stats.OpenConnections >= stats.MaxOpenConnections {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("Health")).Warnz("database connection pool is full",
			zap.Int("open", stats.OpenConnections),
			zap.Int("max", stats.MaxOpenConnections),
			zap.Int("in_use", stats.InUse),
			zap.Int("idle", stats.Idle),
			zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
		)
	}

	// 3.check database response time
	if err := sqlDB.PingContext(context.TODO()); err != nil {
		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("Health")).Errorz("database ping failed",
			zap.Error(err),
			zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
		)
		return fmt.Errorf("database ping failed: %w", err)
	}

	logger.Database.WithDatabaseContext(db.ctx, consts.Phase("Health")).Infoz("database health check passed",
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use_connections", stats.InUse),
		zap.Int("idle_connections", stats.Idle),
		zap.Int("max_open_connections", stats.MaxOpenConnections),
		zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
	)

	return nil
}

// TransactionFunc executes a function within a complete transaction with automatic management.
// This is the recommended method for most transaction scenarios as it provides:
// 1. Automatic transaction begin/commit/rollback management
// 2. Built-in error handling and logging
// 3. Performance monitoring with execution time tracking
// 4. Type-safe transaction context
//
// Transaction behavior:
// - If the function returns nil: transaction is automatically committed
// - If the function returns an error: transaction is automatically rolled back
// - All database operations within the function are executed in the same transaction
//
// Relationship with other transaction methods:
// - Use TransactionFunc: For most transaction scenarios (recommended)
// - Use WithRollback: When you need manual control over rollback timing
//
// Example usage:
//
//	// Simple transaction with automatic management
//	err := database.Database[*model.User](nil).TransactionFunc(func(tx any) error {
//	    userDB := database.Database[*model.User](nil)
//	    if err := userDB.WithTx(tx).Create(&user); err != nil {
//	        return err // Automatic rollback
//	    }
//	    if err := userDB.WithTx(tx).UpdateByID(user.ID, "status", "active"); err != nil {
//	        return err // Automatic rollback
//	    }
//	    return nil // Automatic commit
//	})
//
//	// Complex transaction with multiple operations
//	err := database.Database[*model.Order](nil).TransactionFunc(func(tx any) error {
//	    orderDB := database.Database[*model.Order](nil)
//	    // All operations share the same transaction context
//	    if err := orderDB.WithTx(tx).WithLock(consts.LockUpdate).Get(&order, orderID); err != nil {
//	        return err
//	    }
//	    order.Status = "processed"
//	    return orderDB.WithTx(tx).Update(&order)
//	})
//
// TransactionFunc executes a function within a database transaction.
// The tx parameter is the underlying GORM transaction instance (*gorm.DB).
// Use WithTx(tx) to create database instances that operate within the transaction.
// If the function returns an error, the transaction is rolled back.
// If the function returns nil, the transaction is committed.
// When used with WithRollback, you can call the rollback function directly
// to trigger a manual rollback.
//
// Example with automatic transaction management:
//
//	err := db.TransactionFunc(func(tx any) error {
//	    userDB := database.Database[*model.User](nil)
//	    if err := userDB.WithTx(tx).Create(&user); err != nil {
//	        return err // automatic rollback
//	    }
//	    return nil // automatic commit
//	})
//
// Example with manual rollback control:
//
//	var rollbackFunc func() error
//	err := db.WithRollback(func() error {
//	    // custom rollback logic
//	    return nil
//	}).TransactionFunc(func(tx any) error {
//	    userDB := database.Database[*model.User](nil)
//	    // Get the rollback function from the transaction context
//	    if txDB, ok := userDB.WithTx(tx).(*database[*model.User]); ok && txDB.rollbackFunc != nil {
//	        rollbackFunc = txDB.rollbackFunc
//	    }
//
//	    if err := userDB.WithTx(tx).Create(&user); err != nil {
//	        return err // automatic rollback
//	    }
//
//	    if someCondition {
//	        if rollbackFunc != nil {
//	            rollbackFunc() // execute custom rollback logic
//	        }
//	        return ErrManualRollback // trigger transaction rollback
//	    }
//	    return nil // automatic commit
//	})
func (db *database[M]) TransactionFunc(fn func(tx any) error) error {
	if err := db.prepare(); err != nil {
		return err
	}

	begin := time.Now()

	return db.ins.Transaction(func(tx *gorm.DB) error {
		// Execute the user function with the transaction gorm.DB instance
		if err := fn(tx); err != nil {
			// Check if this is a manual rollback request
			if errors.Is(err, ErrManualRollback) {
				// Execute custom rollback logic if provided
				if db.rollbackFunc != nil {
					if rollbackErr := db.rollbackFunc(); rollbackErr != nil {
						logger.Database.WithDatabaseContext(db.ctx, consts.Phase("TransactionFunc")).Errorz("custom rollback function failed",
							zap.Error(rollbackErr),
							zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
						)
					}
				}
				logger.Database.WithDatabaseContext(db.ctx, consts.Phase("TransactionFunc")).Infoz("transaction rolled back manually",
					zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
				)
			} else {
				logger.Database.WithDatabaseContext(db.ctx, consts.Phase("TransactionFunc")).Errorz("transaction rolled back due to error",
					zap.Error(err),
					zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
				)
			}
			return err
		}

		logger.Database.WithDatabaseContext(db.ctx, consts.Phase("TransactionFunc")).Infoz("transaction committed successfully",
			zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
		)
		return nil
	})
}

// Database creates and returns a generic database manipulator implementing types.Database interface.
// Provides comprehensive CRUD capabilities with advanced features like caching, hooks, and query building.
// Automatically enables debug mode when log level is set to debug.
//
// Type Parameters:
//   - M: Model type that implements types.Model interface
//
// Parameters:
//   - ctx: Required database context for request tracing and metadata.
//     In service layer operations, pass a valid DatabaseContext to track requests.
//     For non-service layer operations, pass nil.
//
// Returns a database manipulator with full CRUD and query capabilities.
//
// Features:
//   - Generic type safety for model operations
//   - Automatic debug mode based on configuration
//   - Context-aware operations for tracing
//   - Default query limit protection
//   - Panic protection for uninitialized database
//
// Example:
//
//	// Service layer usage with context
//	db := Database[*User](ctx.DatabaseContext())
//	// Non-service layer usage
//	db := Database[*User](nil)
//	users := db.WithQuery(&User{Name: "John"}).List()
func Database[M types.Model](ctx *types.DatabaseContext) types.Database[M] {
	if DB == nil || DB == new(gorm.DB) {
		panic("database is not initialized")
	}
	dbctx := new(types.DatabaseContext)
	gctx := context.Background()
	if ctx != nil {
		dbctx = ctx
		gctx = dbctx.Context()
	}

	var ins *gorm.DB
	if strings.ToLower(config.App.Logger.Level) == "debug" {
		ins = DB.Debug().WithContext(gctx).Limit(defaultLimit)
	} else {
		ins = DB.WithContext(gctx).Limit(defaultLimit)
	}

	db := &database[M]{
		ins: ins,
		ctx: dbctx,
	}

	// Set up auto migration for default database if not already migrated
	// Use database identifier + model type as key to support multiple database instances
	dbIdentifier := getDBIdentifier(DB)
	modelType := reflect.TypeFor[M]().String()
	migrationKey := fmt.Sprintf("%s:%s", dbIdentifier, modelType)
	if _, loaded := migratedModelMap.LoadOrStore(migrationKey, struct{}{}); !loaded {
		flag := new(bool)
		*flag = true
		db.shouldAutoMigrate = flag
	}

	return db
}
