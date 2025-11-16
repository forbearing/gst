package types

import (
	"context"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/types/consts"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ErrEntryNotFound is returned when a cache entry is not found.
var ErrEntryNotFound = errors.New("cache entry not found")

// Initializer interface is used to initialize configuration, flag arguments, logger, or other components.
// This interface is commonly implemented by bootstrap components that need to perform
// initialization tasks during application startup.
//
// Example implementations:
//   - Configuration loaders
//   - Logger initializers
//   - Database connection setup
//   - Cache initialization
type Initializer interface {
	Init() error
}

// StandardLogger interface provides standard logging methods for custom logger implementations.
// This interface follows the traditional logging pattern with both simple and formatted logging methods.
//
// Usage:
//   - Implement this interface to create custom loggers
//   - Use Debug/Info/Warn/Error for simple logging
//   - Use Debugf/Infof/Warnf/Errorf for formatted logging
//   - Fatal methods should terminate the program after logging
type StandardLogger interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// StructuredLogger interface provides structured logging methods with key-value pairs.
// This interface is designed for structured logging where additional context can be
// attached to log messages as key-value pairs.
//
// Usage:
//
//	logger.Infow("User login", "userID", 123, "ip", "192.168.1.1")
//	logger.Errorw("Database error", "error", err, "query", sql)
//
// The 'w' suffix stands for "with" (structured data).
type StructuredLogger interface {
	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Fatalw(msg string, keysAndValues ...any)
}

// ZapLogger interface provides zap-specific logging methods with structured fields.
// This interface is designed for integration with the uber-go/zap logging library,
// offering high-performance structured logging capabilities.
//
// Usage:
//
//	logger.Infoz("Request processed", zap.String("method", "GET"), zap.Int("status", 200))
//	logger.Errorz("Database connection failed", zap.Error(err), zap.String("host", dbHost))
//
// The 'z' suffix distinguishes these methods from other logging interfaces.
type ZapLogger interface {
	Debugz(msg string, fields ...zap.Field)
	Infoz(msg string, fields ...zap.Field)
	Warnz(msg string, fields ...zap.Field)
	Errorz(msg string, fields ...zap.Field)
	Fatalz(msg string, fields ...zap.Field)
}

// Logger interface combines all logging capabilities into a unified interface.
// This interface provides comprehensive logging functionality by embedding
// StandardLogger, StructuredLogger, and ZapLogger interfaces, along with
// context-aware logging methods.
//
// Key features:
//   - Standard logging (Debug, Info, Warn, Error, Fatal)
//   - Structured logging with key-value pairs (Debugw, Infow, etc.)
//   - Zap-specific structured logging with typed fields
//   - Context-aware logging for controllers, services, and database operations
//   - Support for complex object and array marshaling
//
// This unified approach allows flexible logging usage throughout the application.
type Logger interface {
	With(fields ...string) Logger

	WithObject(name string, obj zapcore.ObjectMarshaler) Logger
	WithArray(name string, arr zapcore.ArrayMarshaler) Logger

	WithControllerContext(*ControllerContext, consts.Phase) Logger
	WithServiceContext(*ServiceContext, consts.Phase) Logger
	WithDatabaseContext(*DatabaseContext, consts.Phase) Logger

	StandardLogger
	StructuredLogger
	ZapLogger
}

// Database interface provides comprehensive database operations for any model type.
// This interface is constrained by the Model interface, ensuring type safety
// for database operations across different model implementations.
//
// Generic constraint:
//
//	M must implement the Model interface (typically by embedding model.Base)
//
// Core operations:
//   - CRUD operations with automatic timestamp management
//   - Flexible querying with various finder methods
//   - Health monitoring and cleanup capabilities
//   - Optional caching support for improved performance
//
// The interface embeds DatabaseOption[M] to provide chainable query building.
type Database[M Model] interface {
	// Create one or multiple record.
	// Pass M to create one record,
	// Pass []M to create multiple record.
	// It will update the "created_at" and "updated_at" field.
	Create(objs ...M) error
	// Delete one or multiple record.
	// Pass M to delete one record.
	// Pass []M to delete multiple record.
	Delete(objs ...M) error
	// Update one or multiple record, if record doesn't exist, it will be created.
	// Pass M to update one record.
	// Pass []M to update multiple record.
	// It will just update the "updated_at" field.
	Update(objs ...M) error
	// UpdateByID only update one record with specific id.
	// its not invoke model hook.
	UpdateByID(id string, key string, value any) error
	// List all records and write to dest.
	List(dest *[]M, cache ...*[]byte) error
	// Get one record with specific id and write to dest.
	Get(dest M, id string, cache ...*[]byte) error
	// First finds the first record ordered by primary key.
	First(dest M, cache ...*[]byte) error
	// Last finds the last record ordered by primary key
	Last(dest M, cache ...*[]byte) error
	// Take finds the first record returned by the database in no specified order.
	Take(dest M, cache ...*[]byte) error
	// Count returns the total number of records with the given query condition.
	Count(*int64) error
	// Cleanup delete all records that column 'deleted_at' is not null.
	Cleanup() error
	// Health checks the database connectivity and basic operations.
	// It returns nil if the database is healthy, otherwise returns an error.
	Health() error
	// Transaction executes a function within a transaction with automatic context injection.
	// The transaction database instance is automatically provided to the callback function.
	// This is the recommended method for single-model transactions as it prevents forgetting WithTx.
	// If the function returns an error, the transaction is automatically rolled back.
	// If the function completes successfully, the transaction is committed.
	Transaction(fn func(txDB Database[M]) error) error
	// TransactionFunc executes a function within a transaction with manual context management.
	// Use this method when you need to operate on multiple model types within the same transaction.
	// You must manually call WithTx(tx) for each database instance to ensure operations are transactional.
	// If the function returns an error, the transaction is automatically rolled back.
	// If the function completes successfully, the transaction is committed.
	TransactionFunc(fn func(tx any) error) error

	DatabaseOption[M]
}

// DatabaseOption interface.
// WithXXX setting database options.
type DatabaseOption[M Model] interface {
	// WithDB sets the underlying GORM database instance for this database manipulator.
	// Only supports *gorm.DB type. Returns the same instance if invalid input is provided.
	//
	// Examples:
	//
	//	WithDB(customDB).Create(&user)
	//	WithDB(customDB).WithTable("users").List(&users)
	//
	// NOTE: If WithTable is called, auto migration will be disabled.
	WithDB(any) Database[M]

	// WithTx returns a new database manipulator with transaction context.
	// This method allows using an existing transaction to operate on multiple resource types.
	// The tx parameter should be a *gorm.DB transaction instance obtained from TransactionFunc.
	//
	// Supports all CRUD operations and can be chained with other methods.
	//
	// Examples:
	//
	//	// Single resource type
	//	TransactionFunc(func(tx any) error {
	//	    return WithTx(tx).Create(&user)
	//	})
	//
	//	// Multiple resource types
	//	TransactionFunc(func(tx any) error {
	//	    if err := Database[*User](nil).WithTx(tx).Create(&user); err != nil {
	//	        return err
	//	    }
	//	    return Database[*Order](nil).WithTx(tx).Create(&order)
	//	})
	WithTx(tx any) Database[M]

	// WithTable sets the table name for database operations, overriding the default table name.
	// Often used in combination with WithDB method.
	// NOTE: Calling WithTable disables auto migration. Manual migration is required.
	//
	// Examples:
	//
	//	WithTable("custom_users").List(&users)
	//	WithDB(customDB).WithTable("users").Create(&user)
	WithTable(name string) Database[M]

	// WithDebug enables debug mode for database operations, showing detailed SQL queries and execution info.
	// This setting has higher priority than config.App.Logger.Level and overrides the default value (false).
	// Useful for development, debugging, and query optimization.
	//
	// Examples:
	//
	//	WithDebug().Create(&user)
	//	WithDebug().WithQuery(&User{Name: "John"}).List(&users)
	WithDebug() Database[M]

	// WithQuery sets query conditions based on model struct fields.
	// Supports exact matching, fuzzy matching (LIKE/REGEXP), OR/AND logic, and raw SQL queries via QueryConfig.
	// Non-zero fields in the model will be used as query conditions.
	// Query can be nil to indicate empty query (requires AllowEmpty: true to allow full table scan).
	//
	// Examples:
	//
	//	WithQuery(&User{Name: "John"})  // Exact match
	//	WithQuery(&User{Name: "John"}, QueryConfig{FuzzyMatch: true})  // Fuzzy match
	//	WithQuery(&User{Name: "John", Email: "john@example.com"}, QueryConfig{UseOr: true})  // OR logic
	//	WithQuery(nil, QueryConfig{RawQuery: "age > ?", RawQueryArgs: []any{18}})  // Raw SQL
	//	WithQuery(nil, QueryConfig{AllowEmpty: true})  // Empty query (returns all records)
	WithQuery(query M, config ...QueryConfig) Database[M]

	// WithCursor enables cursor-based pagination for efficient large dataset traversal.
	// Cursor pagination is more efficient than offset-based pagination for large datasets
	// as it avoids performance degradation when skipping many records.
	//
	// Parameters:
	//   - cursorValue: The value of the cursor field from the last record in the previous page.
	//     For string fields (like ID), use the field value directly.
	//     For time fields, format as "YYYY-MM-DD HH:MM:SS.ffffff".
	//     Empty string will be ignored and cursor pagination will be disabled.
	//   - next: The direction of pagination.
	//     - true: Fetch records after the cursor (next page, ascending order)
	//     - false: Fetch records before the cursor (previous page, descending order)
	//   - fields: Optional field name(s) to use as cursor. Defaults to "id" if not specified.
	//     Currently only the first field is used (multiple fields support is TODO).
	//
	// Behavior:
	//   - When next=true: Returns records where cursorField > cursorValue, ordered by cursorField ASC
	//   - When next=false: Returns records where cursorField < cursorValue, ordered by cursorField DESC
	//     (results are reversed to maintain original sort order)
	//   - Empty cursorValue: Cursor pagination is disabled, returns all records
	//   - Default cursor field: "id" if not specified
	//
	// Example:
	//
	//	// First page (no cursor)
	//	database.Database[*model.User]().WithLimit(10).List(&users)
	//
	//	// Next page (using last user's ID as cursor)
	//	lastID := users[len(users)-1].ID
	//	database.Database[*model.User]().WithCursor(lastID, true).WithLimit(10).List(&nextUsers)
	//
	//	// Next page using custom field (created_at)
	//	lastCreatedAt := users[len(users)-1].CreatedAt.Format("2006-01-02 15:04:05.000000")
	//	database.Database[*model.User]().WithCursor(lastCreatedAt, true, "created_at").WithLimit(10).List(&nextUsers)
	//
	//	// Previous page
	//	firstID := users[0].ID
	//	database.Database[*model.User]().WithCursor(firstID, false).WithLimit(10).List(&prevUsers)
	WithCursor(string, bool, ...string) Database[M]

	// WithTimeRange applies a time range filter to the query based on the specified column name.
	// It restricts the results to records where the column's value falls within the specified start and end times.
	// This method is designed to be used in a chainable manner, allowing for the construction of complex queries.
	//
	// Parameters:
	// - columnName: The name of the column to apply the time range filter on. This should be a valid date/time column in the database.
	// - startTime: The beginning of the time range. Records with the column's value equal to or later than this time will be included.
	// - endTime: The end of the time range. Records with the column's value equal to or earlier than this time will be included.
	//
	// Returns: A modified Database instance that includes the time range filter in its query conditions.
	WithTimeRange(columnName string, startTime time.Time, endTime time.Time) Database[M]

	// WithSelect specify fields that you want when querying, creating, updating
	// default select all fields.
	WithSelect(columns ...string) Database[M]

	// WithSelectRaw
	WithSelectRaw(query any, args ...any) Database[M]

	// WithIndex specifies database index hints for query optimization.
	// The first parameter is the index name, and the second optional parameter specifies the hint type.
	// If no hint type is provided, defaults to USE INDEX.
	//
	// Parameters:
	//   - indexName: The name of the index to hint. Empty or whitespace-only names are ignored.
	//   - hint: Optional hint mode. If not provided, defaults to consts.IndexHintUse.
	//     Supported modes:
	//     - consts.IndexHintUse: Suggests the database to use the specified index
	//     - consts.IndexHintForce: Forces the database to use the specified index
	//     - consts.IndexHintIgnore: Tells the database to ignore the specified index
	//
	// IMPORTANT: Index hints are ONLY supported in SELECT queries (List, Get, Count, First, Last, Take).
	// They are NOT supported in INSERT, UPDATE, DELETE operations. Using WithIndex with Create, Update,
	// or Delete methods will result in SQL syntax errors.
	//
	// Database Compatibility:
	//   - MySQL: Fully supported. All hint modes work as expected.
	//   - SQLite/PostgreSQL/Other databases: Not supported. The hint is silently ignored.
	//
	// Examples:
	//
	//	// Default USE INDEX hint
	//	WithIndex("idx_name").List(&users)
	//
	//	// Explicit hint modes
	//	WithIndex("idx_name", consts.IndexHintForce).List(&users)
	//	WithIndex("idx_name", consts.IndexHintIgnore).List(&users)
	//
	//	// Combined with other methods
	//	WithIndex("idx_name").WithQuery(&User{Name: "John"}).List(&users)
	//
	// NOTE: Index hints are MySQL-specific. On other databases, the hint is silently ignored.
	// NOTE: Empty or whitespace-only index names are automatically ignored.
	WithIndex(indexName string, hint ...consts.IndexHintMode) Database[M]

	// WithRollback configures a rollback function for manual transaction control.
	// This method should be used with TransactionFunc to enable manual rollback capability.
	WithRollback(rollbackFunc func()) Database[M]

	// WithJoinRaw
	WithJoinRaw(query string, args ...any) Database[M]

	// TODO:
	// WithGroup(name string) Database[M]
	// WithHaving(query any, args ...any) Database[M]

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
	WithLock(mode ...consts.LockMode) Database[M]

	// WithBatchSize sets the batch size for batch operations such as batch insert, update, or delete.
	// Controls how many records are processed in a single database operation to optimize performance.
	//
	// Parameters:
	//   - size: The number of records to process per batch.
	//     If set to 0 or negative, uses default batch sizes:
	//     - Create/Update: 1000 records per batch
	//     - Delete: 10000 records per batch
	//     If set to a positive value, uses that value for all operations.
	//
	// Affects Create, Update, and Delete operations.
	//
	// Examples:
	//
	//	WithBatchSize(1000).Create(users...)
	//	WithBatchSize(500).Update(users...)
	//	WithBatchSize(2000).Delete(users...)
	WithBatchSize(size int) Database[M]

	// WithPagination applies pagination parameters to the query, useful for retrieving data in pages.
	// This method enables front-end applications to request a specific subset of records,
	// based on the desired page number and the number of records per page.
	//
	// Parameters:
	// - page: The page number being requested. Page numbers typically start at 1.
	// - size: The number of records to return per page. This determines the "size" of each page.
	//
	// The pagination logic calculates the offset based on the page number and size,
	// and applies it along with the limit (size) to the query. This facilitates efficient
	// data fetching suitable for front-end pagination displays.
	//
	// Returns: A modified Database instance that includes pagination parameters in its query conditions.
	WithPagination(page, size int) Database[M]

	// WithLimit determines how much record should retrieve.
	// limit is 0 or -1 means no limit.
	WithLimit(limit int) Database[M]

	// WithExclude excludes records that matchs a condition within a list.
	// For example:
	//   - If you want exclude users with specific ids from your query,
	//     you can use WithExclude(excludes),
	//     excludes: "id" as key, ["myid1", "myid2", "myid3"] as value.
	//   - If you want excludes users that id not ["myid1", "myid2"] and not not ["root", "noname"],
	//     the `excludes` should be:
	//     excludes := make(map[string][]any)
	//     excludes["id"] = []any{"myid1", "myid2"}
	//     excludes["name"] = []any{"root", "noname"}.
	WithExclude(map[string][]any) Database[M]

	// WithOrder adds ORDER BY clause to sort query results.
	// Supports multiple sorting criteria and directions (ASC/DESC).
	// Column names are automatically wrapped with backticks to handle SQL keywords.
	//
	// Parameters:
	//   - order: Column name(s) with optional direction. Multiple columns separated by commas.
	//            Direction can be "ASC" (default) or "DESC" (case-insensitive).
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
	WithOrder(order string) Database[M]

	// WithExpand, for "foreign key".
	WithExpand(expand []string, order ...string) Database[M]

	// WithPurge tells the database manipulator to delete resource in database permanently.
	WithPurge(...bool) Database[M]
	// WithCache tells the database manipulator to retrieve resource from cache.
	WithCache(...bool) Database[M]
	// WithOmit omit specific columns when create/update.
	WithOmit(...string) Database[M]
	// WithDryRun enables dry-run mode to preview SQL queries without executing them.
	// Only executes model hooks without performing actual database operations.
	// Also logs the SQL statements that would have been executed.
	WithDryRun() Database[M]
	// WithoutHook tells the database manipulator not invoke model hooks.
	WithoutHook() Database[M]
}

// Model interface defines the contract for all data models in the framework.
// This interface ensures consistent behavior across different model implementations
// and provides comprehensive functionality for database operations, logging, and lifecycle hooks.
//
// Implementation requirements:
//  1. Must be a pointer to a struct (e.g., *User, not User) - otherwise causes panic
//  2. Must have an "ID" field as the primary key in the database
//  3. Should embed model.Base to inherit common fields and methods
//
// Example implementation:
//
//	type User struct {
//	    model.Base
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	func (u *User) GetTableName() string {
//	    return "users"
//	}
//
// Core functionality:
//   - Table and ID management for database operations
//   - Audit trail with created/updated timestamps and user tracking
//   - Relationship management through Expands() for foreign key preloading
//   - Query filtering through Excludes() for conditional operations
//   - Structured logging support via zap.ObjectMarshaler
//   - Lifecycle hooks for custom business logic during CRUD operations
type Model interface {
	GetTableName() string // GetTableName returns the table name.
	GetID() string
	SetID(id ...string) // SetID method will automatically set the id if id is empty.
	ClearID()           // ClearID always set the id to empty.
	GetCreatedBy() string
	GetUpdatedBy() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	SetCreatedBy(string)
	SetUpdatedBy(string)
	SetCreatedAt(time.Time)
	SetUpdatedAt(time.Time)
	Expands() []string // Expands returns the foreign keys should preload.
	Excludes() map[string][]any
	Purge() bool                                  // Purge indicates whether to permanently delete records (hard delete). Default is false (soft delete).
	MarshalLogObject(zapcore.ObjectEncoder) error // MarshalLogObject implement zap.ObjectMarshaler

	CreateBefore(*ModelContext) error
	CreateAfter(*ModelContext) error
	DeleteBefore(*ModelContext) error
	DeleteAfter(*ModelContext) error
	UpdateBefore(*ModelContext) error
	UpdateAfter(*ModelContext) error
	ListBefore(*ModelContext) error
	ListAfter(*ModelContext) error
	GetBefore(*ModelContext) error
	GetAfter(*ModelContext) error
}

type (
	Request  any
	Response any
)

// Service interface provides comprehensive business logic operations for model types.
// This interface defines the service layer that sits between controllers and database operations,
// implementing business rules, validation, complex operations, and lifecycle management.
//
// Implementation requirements:
//   - The implementing object must be a pointer to struct
//
// Generic constraints:
//   - M: Must implement the Model interface
//   - REQ: Request type (typically DTOs or request structures)
//   - RSP: Response type (typically DTOs or response structures)
//
// Core operations:
//   - CRUD operations: Create, Delete, Update, Patch, List, Get
//   - Batch operations: CreateMany, DeleteMany, UpdateMany, PatchMany
//   - Lifecycle hooks: Before/After methods for each operation
//   - Data operations: Import/Export for bulk data management
//   - Filtering: Custom filtering logic for queries
//
// ServiceContext provides:
//   - HTTP request/response context
//   - Database transaction management
//   - User authentication and authorization context
//   - Request validation and data binding
//   - Logging and tracing capabilities
//
// Hook methods allow custom business logic:
//   - Before hooks: Validation, authorization, data transformation
//   - After hooks: Notifications, caching, audit logging, cleanup
type Service[M Model, REQ Request, RSP Response] interface {
	Create(*ServiceContext, REQ) (RSP, error)
	Delete(*ServiceContext, REQ) (RSP, error)
	Update(*ServiceContext, REQ) (RSP, error)
	Patch(*ServiceContext, REQ) (RSP, error)
	List(*ServiceContext, REQ) (RSP, error)
	Get(*ServiceContext, REQ) (RSP, error)

	CreateMany(*ServiceContext, REQ) (RSP, error)
	DeleteMany(*ServiceContext, REQ) (RSP, error)
	UpdateMany(*ServiceContext, REQ) (RSP, error)
	PatchMany(*ServiceContext, REQ) (RSP, error)

	CreateBefore(*ServiceContext, M) error
	CreateAfter(*ServiceContext, M) error
	DeleteBefore(*ServiceContext, M) error
	DeleteAfter(*ServiceContext, M) error
	UpdateBefore(*ServiceContext, M) error
	UpdateAfter(*ServiceContext, M) error
	PatchBefore(*ServiceContext, M) error
	PatchAfter(*ServiceContext, M) error
	ListBefore(*ServiceContext, *[]M) error
	ListAfter(*ServiceContext, *[]M) error
	GetBefore(*ServiceContext, M) error
	GetAfter(*ServiceContext, M) error

	CreateManyBefore(*ServiceContext, ...M) error
	CreateManyAfter(*ServiceContext, ...M) error
	DeleteManyBefore(*ServiceContext, ...M) error
	DeleteManyAfter(*ServiceContext, ...M) error
	UpdateManyBefore(*ServiceContext, ...M) error
	UpdateManyAfter(*ServiceContext, ...M) error
	PatchManyBefore(*ServiceContext, ...M) error
	PatchManyAfter(*ServiceContext, ...M) error

	Import(*ServiceContext, io.Reader) ([]M, error)
	Export(*ServiceContext, ...M) ([]byte, error)

	Filter(*ServiceContext, M) M
	FilterRaw(*ServiceContext) string

	Logger
}

// Cache interface provides a unified caching abstraction with consistent error handling.
// This interface supports various cache operations with proper error reporting and
// distributed tracing capabilities.
//
// Generic type T can be any serializable data type.
//
// Error Handling:
//
//	All operations return an error to provide comprehensive error information.
//	For Get/Peek operations, ErrEntryNotFound is returned when the key doesn't exist.
//	This design follows Go best practices and aligns with standard library patterns.
//
// Operations:
//   - Set: Store value with TTL, returns error on failure
//   - Get: Retrieve value and mark as accessed, returns ErrEntryNotFound if key doesn't exist
//   - Peek: Retrieve value without affecting access order, returns ErrEntryNotFound if key doesn't exist
//   - Delete: Remove specific key, returns error on failure
//   - Exists: Check if key exists, returns bool
//   - Len: Get current number of cached items, returns int
//   - Clear: Remove all cached items
//   - WithContext: Returns cache instance with tracing context for distributed tracing
//
// Key features:
//   - Type-safe operations with generics
//   - Consistent error handling across all operations
//   - TTL support for expirable entries
//   - Context-aware operations for tracing
//   - Thread-safe operations
//
// Error handling:
//   - Returns ErrEntryNotFound when cache entries are not found
//   - All operations return errors for proper error handling
//   - Supports graceful degradation in distributed environments
type Cache[T any] interface {
	// Get retrieves a value from the cache by key.
	// Returns ErrEntryNotFound if the key does not exist.
	Get(key string) (T, error)

	// Peek retrieves a value from the cache by key without affecting its position or access time.
	// Returns ErrEntryNotFound if the key does not exist.
	Peek(key string) (T, error)

	// Set stores a value in the cache with the specified TTL.
	// A zero TTL means the entry will not expire.
	Set(key string, value T, ttl time.Duration) error

	// Delete removes a key from the cache.
	// Returns ErrEntryNotFound if the key does not exist.
	Delete(key string) error

	// Exists checks if a key exists in the cache.
	// Returns true if the key exists, false otherwise.
	Exists(key string) bool

	// Len returns the number of entries currently stored in the cache.
	Len() int

	// Clear removes all entries from the cache.
	Clear()

	// WithContext replaces the cache internal context that used to propagate span context.
	WithContext(ctx context.Context) Cache[T]
}

// DistributedCache defines a two-level distributed caching system that combines local memory cache
// with Redis backend for high-performance, synchronized caching across multiple nodes.
//
// Architecture:
//   - Local Cache: High-speed in-memory cache for immediate access
//   - Redis Cache: Distributed persistent storage for cross-node data sharing
//   - Kafka Events: Real-time cache synchronization and invalidation across nodes
//   - State Node: Coordinates cache operations and ensures consistency
//
// Key Features:
//   - Automatic cache synchronization across multiple application instances
//   - Configurable TTL for both local and distributed cache layers
//   - Event-driven cache invalidation using Kafka messaging
//   - Performance metrics tracking (hits, misses, operations)
//   - Goroutine pool for efficient concurrent operations
//   - Type-safe generic implementation
//
// Usage Patterns:
//   - Use SetWithSync/GetWithSync/DeleteWithSync for data that needs cross-node synchronization
//   - Use regular Cache[T] methods (Set/Get/Delete) for local-only operations
//   - Configure appropriate TTL values: localTTL <= remoteTTL for optimal performance
//
// Thread Safety:
//   - All operations are thread-safe and can be called concurrently
//   - Internal synchronization handles concurrent access to cache maps
//
// Performance Considerations:
//   - Local cache provides sub-microsecond access times
//   - Redis operations add network latency but ensure data consistency
//   - Kafka events enable near real-time cache synchronization
//   - Goroutine pool prevents resource exhaustion under high load
type DistributedCache[T any] interface {
	Cache[T]

	// SetWithSync stores a value in both local and distributed cache with synchronization.
	//
	// Operation flow:
	//	1. Set value in local cache with localTTL expiration
	//	2. Send 'Set' event to state node
	//	3. State node sets Redis cache with remoteTTL expiration (Cache.Set method does not set Redis cache)
	//	4. State node sends 'SetDone' event
	//	5. Current node updates local cache
	SetWithSync(key string, value T, localTTL time.Duration, remoteTTL time.Duration) error

	// GetWithSync retrieves a value from local cache first, then from distributed cache if not found.
	//
	// Operation flow:
	//	1. Retrieve from local cache
	//	2. If not found in local cache, retrieve from Redis
	//	3. If found in Redis, backfill to local cache with localTTL expiration
	//	   Note: Backfilling local cache does not send 'Set' event to state node
	GetWithSync(key string, localTTL time.Duration) (T, error)

	// DeleteWithSync removes a value from both local and distributed cache with synchronization.
	//
	// Operation flow:
	//	1. Delete from local cache
	//	2. Send 'Del' event to state node
	//	3. State node deletes Redis cache (Cache.Delete method does not delete Redis cache)
	//	4. State node sends 'DelDone' event
	//	5. Current node deletes from local cache
	DeleteWithSync(key string) error
}

// RBAC interface defines comprehensive role-based access control operations.
// This interface provides a complete RBAC system supporting roles, permissions,
// and subject assignments with flexible resource and action management.
//
// RBAC Model Components:
//   - Subject: Users or entities that need access (e.g., "user:123", "service:api")
//   - Role: Named collection of permissions (e.g., "admin", "editor", "viewer")
//   - Resource: Protected objects or endpoints (e.g., "users", "posts", "/api/v1/users")
//   - Action: Operations on resources (e.g., "read", "write", "delete", "create")
//
// Permission Model:
//   - Permissions are defined as (role, resource, action) tuples
//   - Subjects are assigned roles, inheriting all role permissions
//   - Supports hierarchical roles and resource patterns
//
// Implementation:
//   - Typically backed by Casbin for policy enforcement
//   - Supports both file-based and database-backed policy storage
//   - Can integrate with external identity providers
//
// Usage patterns:
//   - API endpoint authorization
//   - Resource-level access control
//   - Multi-tenant permission management
type RBAC interface {
	AddRole(name string) error
	RemoveRole(name string) error

	GrantPermission(role string, resource string, action string) error
	RevokePermission(role string, resource string, action string) error

	AssignRole(subject string, role string) error
	UnassignRole(subject string, role string) error
}

// Module interface defines a comprehensive module system for creating modular API endpoints
// with automatic CRUD operations, routing, and service layer integration.
//
// The Module interface enables developers to create reusable, self-contained modules
// that automatically register models, services, and routes with the framework.
// Each module encapsulates a complete API resource with customizable behavior.
//
// Generic constraints:
//   - M: Must implement the Model interface (typically a pointer to struct)
//   - REQ: Request type for API operations (can be any serializable type)
//   - RSP: Response type for API operations (can be any serializable type)
//
// Implementation requirements:
//  1. Must provide a Service instance that handles business logic
//  2. Must define the API route path for the resource
//  3. Can optionally customize URL parameter names
//  4. Can optionally make endpoints public (bypass authentication)
//
// Usage example:
//
//	type HelloworldModule struct{}
//
//	func (HelloworldModule) Service() types.Service[*Helloworld, *Req, *Rsp] {
//	    return &HelloworldService{}
//	}
//	func (HelloworldModule) Pub() bool     { return false }  // requires auth
//	func (HelloworldModule) Route() string { return "hello-world" }
//	func (HelloworldModule) Param() string { return "id" }   // URL param name
//
//	// Register the module with specific CRUD phases
//	module.Use[*Helloworld, *Req, *Rsp, *HelloworldService](
//	    &HelloworldModule{},
//	    consts.PHASE_CREATE,
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	)
//
// This will automatically create the following routes:
//   - POST   /hello-world        (create)
//   - GET    /hello-world        (list)
//   - GET    /hello-world/:id    (get by id)
//
// Authentication behavior:
//   - If Pub() returns true: endpoints are publicly accessible
//   - If Pub() returns false: endpoints require authentication/authorization
type Module[M Model, REQ Request, RSP Response] interface {
	// Service returns the service instance that handles business logic for this module.
	// The service must implement all CRUD operations and lifecycle hooks.
	Service() Service[M, REQ, RSP]

	// Pub determines whether the API endpoints are public or require authentication.
	// Returns true for public endpoints, false for authenticated endpoints.
	// Default behavior should return false (require authentication).
	Pub() bool

	// Route returns the base API path for this module's endpoints.
	// The path should not include leading/trailing slashes or "api" prefix.
	// Example: "users", "hello-world", "products"
	Route() string

	// Param returns the URL parameter name used for resource identification.
	// If empty string is returned, defaults to "id".
	// This parameter is used in routes like: GET /resource/:param
	// Example: "id", "uuid", "slug"
	Param() string
}

// ESDocumenter represents a document that can be indexed into Elasticsearch.
// Types implementing this interface should be able to convert themselves
// into a document format suitable for Elasticsearch indexing.
type ESDocumenter interface {
	// Document returns a map representing an Elasticsearch document.
	// The returned map should contain all fields to be indexed, where:
	//   - keys are field names (string type)
	//   - values are field values (any type)
	//
	// Implementation notes:
	//   1. The returned map should only contain JSON-serializable values.
	//   2. Field names should match those defined in the Elasticsearch mapping.
	//   3. Complex types (like nested objects or arrays) should be correctly
	//      represented in the returned map.
	//
	// Example:
	//   return map[string]any{
	//       "id":    "1234",
	//       "title": "Sample Document",
	//       "tags":  []string{"tag1", "tag2"},
	//   }
	Document() map[string]any

	// GetID returns a string that uniquely identifies the document.
	// This ID is typically used as the Elasticsearch document ID.
	//
	// Implementation notes:
	//   1. The ID should be unique within the index.
	//   2. If no custom ID is needed, consider returning an empty string
	//      to let Elasticsearch auto-generate an ID.
	//   3. The ID should be a string, even if it's originally a numeric value.
	//
	// Example:
	//   return "user_12345"
	GetID() string
}
