package database

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/logger"
	"github.com/forbearing/gst/provider/otel"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// trace returns a timing function for database operations that provides comprehensive
// performance monitoring, logging, and distributed tracing capabilities.
// The returned function should be called with the operation result to complete tracing and logging.
//
// Parameters:
//   - op: Operation name for logging and tracing identification (Create, List, Update, Delete, etc.)
//   - batch: Optional batch size for batch operations (used for span attributes and logging)
//
// Returns a function that accepts an error and completes the operation tracing and logging.
//
// Features:
//   - Automatic timing measurement from call to completion
//   - OTEL distributed tracing integration with OpenTelemetry spans
//   - Comprehensive span attributes including operation metadata
//   - Error-aware logging and span status management
//   - Batch operation support with size tracking
//   - Cache and try-run mode status recording
//   - Smart duration formatting for readability
//   - Context propagation to GORM operations
//
// OTEL Tracing Integration:
//   - Creates OpenTelemetry spans with naming pattern: "Database.{Operation} {ModelName}"
//   - Records detailed span attributes: component, operation, model, table, batch_size, etc.
//   - Propagates span context to GORM operations for complete tracing hierarchy
//   - Automatically handles span lifecycle (creation, attribute setting, completion)
//   - Integrates with existing tracing infrastructure (controller and service layers)
//   - Ensures trace_id is available in database logs by backfilling DatabaseContext.TraceID
//     from the span context when missing
//
// Usage Pattern:
//
//	done := db.trace("Create", len(models))
//	defer done(err)
//
// Tracing Hierarchy:
//
//	HTTP → Controller → Service → Database → GORM
//
// Note: Must be called after `defer db.reset()` to ensure proper cleanup order.
// Jaeger tracing is automatically enabled when otel.IsEnabled() returns true.
func (db *database[M]) trace(op string, batch ...int) (func(error), context.Context, trace.Span) {
	begin := time.Now()
	var _batch int
	if len(batch) > 0 {
		_batch = batch[0]
	}

	// Create database operation span if Jaeger is enabled
	var ctx context.Context
	var span trace.Span
	if otel.IsEnabled() && db.ctx != nil {
		modelName := reflect.TypeOf(*new(M)).Elem().Name()
		spanName := "Database." + op + " " + modelName
		ctx, span = otel.StartSpan(db.ctx.Context(), spanName)

		// Propagate OTEL trace ID to DatabaseContext so database logs carry trace_id
		if len(db.ctx.TraceID) == 0 {
			if sc := span.SpanContext(); sc.HasTraceID() {
				db.ctx.TraceID = sc.TraceID().String()
			}
		}

		// Update GORM database context with new span context
		db.ins = db.ins.WithContext(ctx)

		// Add database-specific attributes
		span.SetAttributes(
			attribute.String("component", "database"),
			attribute.String("database.operation", op),
			attribute.String("database.model", modelName),
			attribute.String("database.table", modelName),
		)

		if _batch > 0 {
			span.SetAttributes(attribute.Int("database.batch_size", _batch))
		}

		span.SetAttributes(
			attribute.Bool("database.cache_enabled", db.enableCache),
			attribute.Bool("database.dry_run", db.dryRun),
		)
	}

	return func(err error) {
		// Record duration
		duration := time.Since(begin)

		// Update span with results if available
		if span != nil && span.IsRecording() {
			span.SetAttributes(attribute.Int64("database.duration_ms", duration.Milliseconds()))

			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				otel.RecordError(span, err)
				span.SetAttributes(attribute.Bool("error", true))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			span.End()
		}

		// Log operation results
		if err != nil {
			logger.Database.WithDatabaseContext(db.ctx, consts.Phase(op)).Errorz("",
				zap.Error(err),
				zap.String("table", reflect.TypeOf(*new(M)).Elem().Name()),
				zap.String("batch", strconv.Itoa(_batch)),
				zap.String("cost", util.FormatDurationSmart(duration)),
				zap.Bool("cache_enabled", db.enableCache),
				zap.Bool("dry_run", db.dryRun),
			)
		} else {
			logger.Database.WithDatabaseContext(db.ctx, consts.Phase(op)).Infoz("",
				zap.String("table", reflect.TypeOf(*new(M)).Elem().Name()),
				zap.String("batch", strconv.Itoa(_batch)),
				zap.String("cost", util.FormatDurationSmart(time.Since(begin))),
				zap.Bool("cache_enabled", db.enableCache),
				zap.Bool("dry_run", db.dryRun),
			)
		}
	}, ctx, span
}

// // User returns a generic database manipulator with the `curd` capabilities
// // for *model.User to create/delete/update/list/get in database.
// // The database type deponds on the value of config.Server.DBType.
// func User(ctx ...context.Context) types.Database[*model.User] {
// 	c := context.TODO()
// 	if len(ctx) > 0 {
// 		if ctx[0] != nil {
// 			c = ctx[0]
// 		}
// 	}
// 	if strings.ToLower(config.App.LogLevel) == "debug" {
// 		return &database[*model.User]{db: DB.WithContext(c).Debug().Limit(defaultLimit)}
// 	}
// 	return &database[*model.User]{db: DB.WithContext(c).Limit(defaultLimit)}
// }

// buildCacheKey constructs Redis cache keys for database operations.
// Generates both prefix and full key based on GORM statement and operation type.
// Uses consistent naming convention for cache key organization and collision avoidance.
//
// Parameters:
//   - stmt: GORM statement containing SQL and table information
//   - action: Operation type ("get", "list", "count", etc.)
//   - id: Optional ID for get operations to create simpler keys
//
// Returns prefix, table name and full cache key for Redis operations.
//
// Key Structure:
//   - Prefix: namespace:table_name
//   - Full Key: namespace:table_name:action:identifier
//   - Get operations with ID: namespace:table_name:get:id_value
//   - Other operations: namespace:table_name:action:sql_statement
//
// Features:
//   - Namespace isolation for multi-tenant applications
//   - Table-based key organization
//   - Operation-specific key generation
//   - SQL statement-based cache invalidation
//
// Reference: https://gorm.io/docs/sql_builder.html
func buildCacheKey(stmt *gorm.Statement, action string, id ...string) (prefix, table, key string) {
	prefix = strings.Join([]string{config.App.Redis.Namespace, stmt.Table}, ":")
	table = stmt.Table
	switch strings.ToLower(action) {
	case "get":
		if len(id) > 0 {
			key = strings.Join([]string{config.App.Redis.Namespace, stmt.Table, action, id[0]}, ":")
		} else {
			key = strings.Join([]string{config.App.Redis.Namespace, stmt.Table, action, stmt.SQL.String()}, ":")
		}
	default:
		key = strings.Join([]string{config.App.Redis.Namespace, stmt.Table, action, stmt.SQL.String()}, ":")
	}
	return prefix, table, key
}

// boolToInt converts a boolean value to an integer.
// Returns 1 for true, 0 for false.
// Useful for database operations that require integer representations of boolean values.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// traceModelHook traces model hook execution with OpenTelemetry spans.
// Creates a span for the hook execution and records timing and error information.
//
// Parameters:
//   - ctx: Database context for span creation
//   - hookName: Name of the hook being executed (CreateBefore, CreateAfter, etc.)
//   - modelName: Name of the model type
//   - fn: Hook function to execute
//
// Returns error from hook execution, with span automatically completed.
//
// Features:
//   - Automatic span creation with naming pattern: "Hook.{HookName} {ModelName}"
//   - Records hook execution timing and success/failure status
//   - Integrates with existing tracing infrastructure
//   - Error recording and span status management
//
// Usage Pattern:
//
//	err := traceModelHook(db.ctx, "CreateBefore", "User", func() error {
//		return obj.CreateBefore()
//	})
func traceModelHook[M types.Model](ctx *types.DatabaseContext, phase consts.Phase, parentSpan trace.Span, fn func(ctx context.Context) error) error {
	if !otel.IsEnabled() || ctx == nil || parentSpan == nil {
		return fn(context.Background())
	}

	modelName := reflect.TypeOf(*new(M)).Elem().Name()
	// Create child span under database span for hook execution
	spanName := "Model." + phase.MethodName() + " " + modelName
	parentCtx := trace.ContextWithSpan(context.Background(), parentSpan)
	childCtx, span := otel.StartSpan(parentCtx, spanName)
	defer span.End()

	// Add hook-specific attributes
	span.SetAttributes(
		attribute.String("component", "model"),
		attribute.String("model.model", modelName),
		attribute.String("model.phase", phase.MethodName()),
	)

	// Record start time
	start := time.Now()

	// Execute hook function
	err := fn(childCtx)

	// Record execution results
	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int64("model.duration_ms", duration.Milliseconds()),
		attribute.Bool("model.success", err == nil),
	)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		otel.RecordError(span, err)
		span.SetAttributes(attribute.Bool("error", true))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// func traceModelHook[M types.Model](ctx *types.DatabaseContext, phase consts.Phase, parentSpan trace.Span, fn func() error) error {
// 	if !otel.IsEnabled() || ctx == nil || parentSpan == nil {
// 		return fn()
// 	}
//
// 	modelName := reflect.TypeOf(*new(M)).Elem().Name()
// 	// Create child span under database span for hook execution
// 	spanName := "Model." + phase.MethodName() + " " + modelName
// 	parentCtx := trace.ContextWithSpan(context.Background(), parentSpan)
// 	_, span := otel.StartSpan(parentCtx, spanName)
// 	defer span.End()
//
// 	// Add hook-specific attributes
// 	span.SetAttributes(
// 		attribute.String("component", "model"),
// 		attribute.String("model.model", modelName),
// 		attribute.String("model.phase", phase.MethodName()),
// 	)
//
// 	// Record start time
// 	start := time.Now()
//
// 	// Execute hook function
// 	err := fn()
//
// 	// Record execution results
// 	duration := time.Since(start)
// 	span.SetAttributes(
// 		attribute.Int64("model.duration_ms", duration.Milliseconds()),
// 		attribute.Bool("model.success", err == nil),
// 	)
//
// 	if err != nil {
// 		span.SetStatus(codes.Error, err.Error())
// 		otel.RecordError(span, err)
// 		span.SetAttributes(attribute.Bool("error", true))
// 	} else {
// 		span.SetStatus(codes.Ok, "")
// 	}
//
// 	return err
// }

// contains checks if a string item exists in a string slice.
// Uses a map-based approach for O(n) time complexity with O(n) space complexity.
// More efficient than linear search for larger slices.
//
// Parameters:
//   - slice: The string slice to search in
//   - item: The string item to search for
//
// Returns true if the item is found, false otherwise.
func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	_, ok := set[item]
	return ok
}

// indirectTypeAndValue recursively dereferences pointer types and values.
// Follows pointer chains until reaching a non-pointer type.
// Used for reflection operations that need to work with the underlying concrete type.
//
// Parameters:
//   - t: The reflect.Type to dereference
//   - v: The reflect.Value to dereference
//
// Returns:
//   - reflect.Type: The final non-pointer type
//   - reflect.Value: The final non-pointer value
//   - bool: true if successful, false if encountered nil pointer
//
// Example:
//   - Input: **int (pointer to pointer to int)
//   - Output: int (the underlying int type)
func indirectTypeAndValue(t reflect.Type, v reflect.Value) (reflect.Type, reflect.Value, bool) {
	for t.Kind() == reflect.Pointer {
		if v.IsNil() {
			return t, v, false
		}
		t = t.Elem()
		v = v.Elem()
	}
	return t, v, true
}

// getDBIdentifier returns a unique identifier for the database instance.
// It uses the pointer address of the underlying database connection to distinguish different database instances.
func getDBIdentifier(db *gorm.DB) string {
	if db == nil {
		return "nil"
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB == nil {
		// Fallback to gorm.DB pointer address if we can't get the underlying database connection
		return fmt.Sprintf("%p", db)
	}
	return fmt.Sprintf("%p", sqlDB)
}
