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

// Database provides comprehensive database operations for any model type.
// Supports CRUD operations, flexible querying, transactions, and advanced features.
//
// Type Parameters:
//   - M: Model type that implements Model interface
//
// Features:
//   - CRUD operations with automatic timestamp management
//   - Flexible querying with various finder methods
//   - Transaction support for single and multi-model operations
//   - Health monitoring and cleanup capabilities
//   - Optional caching support for improved performance
//
// The interface embeds DatabaseOption[M] to provide chainable query building.
type Database[M Model] interface {
	// Create inserts one or multiple records into the database.
	Create(objs ...M) error
	// Delete removes one or multiple records from the database.
	Delete(objs ...M) error
	// Update modifies one or multiple records in the database.
	Update(objs ...M) error
	// UpdateByID updates a single field of a record by its ID.
	UpdateByID(id string, key string, value any) error
	// List retrieves multiple records matching the query conditions.
	List(dest *[]M) error
	// Get retrieves a single record by its ID.
	Get(dest M, id string) error
	// First retrieves the first record ordered by primary key.
	First(dest M) error
	// Last retrieves the last record ordered by primary key.
	Last(dest M) error
	// Take retrieves the first record in no specified order.
	Take(dest M) error
	// Count returns the total number of records matching the query conditions.
	Count(*int64) error
	// Cleanup permanently deletes all soft-deleted records.
	Cleanup() error
	// Health checks the database connectivity and basic operations.
	Health() error
	// Transaction executes a function within a transaction (single-model, recommended).
	Transaction(fn func(txDB Database[M]) error) error
	// TransactionFunc executes a function within a transaction (multi-model, requires WithTx).
	TransactionFunc(fn func(tx any) error) error

	DatabaseOption[M]
}

// DatabaseOption provides chainable query building methods for database operations.
// All methods return Database[M] to support method chaining.
type DatabaseOption[M Model] interface {
	// WithDB sets a custom database instance for operations.
	WithDB(any) Database[M]
	// WithTx sets transaction context for operations (used with TransactionFunc).
	WithTx(tx any) Database[M]
	// WithTable sets a custom table name for operations.
	WithTable(name string) Database[M]
	// WithDebug enables debug mode to show detailed SQL queries.
	WithDebug() Database[M]
	// WithQuery sets query conditions based on model fields or raw SQL.
	WithQuery(query M, config ...QueryConfig) Database[M]
	// WithCursor enables cursor-based pagination for efficient large dataset traversal.
	WithCursor(string, bool, ...string) Database[M]
	// WithTimeRange applies a time range filter to the query.
	WithTimeRange(columnName string, startTime time.Time, endTime time.Time) Database[M]
	// WithSelect specifies fields to select in queries.
	WithSelect(columns ...string) Database[M]
	// WithSelectRaw specifies raw SQL for field selection.
	WithSelectRaw(query any, args ...any) Database[M]
	// WithIndex specifies database index hints for query optimization (MySQL only).
	WithIndex(indexName string, hint ...consts.IndexHintMode) Database[M]
	// WithRollback configures a rollback function for manual transaction control.
	WithRollback(rollbackFunc func()) Database[M]
	// WithJoinRaw adds a raw JOIN clause to the query.
	WithJoinRaw(query string, args ...any) Database[M]
	// WithLock adds row-level locking to SELECT queries (must be used within a transaction).
	WithLock(mode ...consts.LockMode) Database[M]
	// WithBatchSize sets the batch size for bulk operations.
	WithBatchSize(size int) Database[M]
	// WithPagination applies pagination parameters (page, size) to the query.
	WithPagination(page, size int) Database[M]
	// WithLimit restricts the number of returned records.
	WithLimit(limit int) Database[M]
	// WithExclude excludes records matching specified conditions.
	WithExclude(map[string][]any) Database[M]
	// WithOrder adds ORDER BY clause to sort query results.
	WithOrder(order string) Database[M]
	// WithExpand enables eager loading of specified associations.
	WithExpand(expand []string, order ...string) Database[M]
	// WithPurge controls whether to permanently delete records (hard delete).
	WithPurge(...bool) Database[M]
	// WithCache enables query result caching.
	WithCache(...bool) Database[M]
	// WithOmit excludes specified fields from operations.
	WithOmit(...string) Database[M]
	// WithDryRun enables dry-run mode to preview SQL without executing.
	WithDryRun() Database[M]
	// WithoutHook disables model hooks for the operation.
	WithoutHook() Database[M]
}

// Model defines the contract for all data models in the framework.
// Provides database operations, audit trail, lifecycle hooks, and logging support.
//
// Type Requirements:
//   - Must be a pointer to struct (e.g., *User)
//   - Must have an "ID" field as primary key
//   - Should embed model.Base for common functionality
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

// Service provides business logic operations for model types.
// Defines the service layer between controllers and database operations.
//
// Type Parameters:
//   - M: Model type that implements Model interface
//   - REQ: Request type (typically DTOs or request structures)
//   - RSP: Response type (typically DTOs or response structures)
//
// Features:
//   - CRUD and batch operations
//   - Lifecycle hooks (Before/After methods)
//   - Data import/export
//   - Custom filtering logic
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

// Cache provides a unified caching abstraction with consistent error handling.
// Supports TTL, context-aware operations, and distributed tracing.
//
// Type Parameters:
//   - T: Serializable data type
//
// Error Handling:
//   - Get/Peek return ErrEntryNotFound when key doesn't exist
//   - All operations return errors for proper error handling
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

// DistributedCache provides a two-level distributed caching system combining local memory cache
// with Redis backend for synchronized caching across multiple nodes.
//
// Type Parameters:
//   - T: Serializable data type
//
// Features:
//   - Automatic cache synchronization across multiple application instances
//   - Configurable TTL for both local and distributed cache layers
//   - Event-driven cache invalidation using Kafka messaging
//   - Thread-safe concurrent operations
type DistributedCache[T any] interface {
	Cache[T]

	// SetWithSync stores a value in both local and distributed cache with synchronization.
	SetWithSync(key string, value T, localTTL time.Duration, remoteTTL time.Duration) error

	// GetWithSync retrieves a value from local cache first, then from distributed cache if not found.
	GetWithSync(key string, localTTL time.Duration) (T, error)

	// DeleteWithSync removes a value from both local and distributed cache with synchronization.
	DeleteWithSync(key string) error
}

// RBAC provides role-based access control operations.
// Supports roles, permissions, and subject assignments with flexible resource and action management.
//
// RBAC Model:
//   - Subject: Users or entities that need access
//   - Role: Named collection of permissions
//   - Resource: Protected objects or endpoints
//   - Action: Operations on resources
type RBAC interface {
	AddRole(name string) error
	RemoveRole(name string) error

	GrantPermission(role string, resource string, action string) error
	RevokePermission(role string, resource string, action string) error

	AssignRole(subject string, role string) error
	UnassignRole(subject string, role string) error
}

// Module defines a module system for creating modular API endpoints
// with automatic CRUD operations, routing, and service layer integration.
//
// Type Parameters:
//   - M: Model type that implements Model interface
//   - REQ: Request type for API operations
//   - RSP: Response type for API operations
//
// Features:
//   - Automatic route registration
//   - Service layer integration
//   - Configurable authentication
type Module[M Model, REQ Request, RSP Response] interface {
	// Service returns the service instance that handles business logic for this module.
	Service() Service[M, REQ, RSP]

	// Pub determines whether the API endpoints are public or require authentication.
	Pub() bool

	// Route returns the base API path for this module's endpoints.
	Route() string

	// Param returns the URL parameter name used for resource identification.
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
