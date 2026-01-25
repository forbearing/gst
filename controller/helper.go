package controller

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/provider/otel"
	. "github.com/forbearing/gst/response"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func patchValue(log types.Logger, typ reflect.Type, oldVal reflect.Value, newVal reflect.Value) {
	for i := range typ.NumField() {
		// fmt.Println(typ.Field(i).Name, typ.Field(i).Type, typ.Field(i).Type.Kind(), newVal.Field(i).IsValid(), newVal.Field(i).CanSet())
		switch typ.Field(i).Type.Kind() {
		case reflect.Struct: // skip update base model.
			switch typ.Field(i).Type.Name() {
			case "GormTime": // The underlying type of model.GormTime(type of time.Time) is struct, we should continue handle.

			case "Base":
				// 有些结构体会匿名继承其他的结构体，例如 AssetChecking 匿名继承 Asset, 所以要可以额外检查是不是某个匿名结构体.
				// 可以自动深度查找,不需要链式查找, 例如
				// newVal.FieldByName("Asset").FieldByName("Remark").IsValid() 可以简化为
				// newVal.FieldByName("Remark").IsValid()

				// Make sure the type of "Remark" is pointer to golang base type.
				fieldRemark := "Remark"
				if oldVal.FieldByName(fieldRemark).CanSet() {
					if newVal.FieldByName(fieldRemark).IsValid() { // WARN: oldVal.FieldByName(fieldRemark) maybe <invalid reflect.Value>
						if !newVal.FieldByName(fieldRemark).IsZero() {
							// output log must before set value.
							if newVal.FieldByName(fieldRemark).Kind() == reflect.Pointer {
								var oldValue, newValue any
								if !oldVal.FieldByName(fieldRemark).IsNil() {
									oldValue = oldVal.FieldByName(fieldRemark).Elem().Interface()
								} else {
									oldValue = "<nil>"
								}
								if !newVal.FieldByName(fieldRemark).IsNil() {
									newValue = newVal.FieldByName(fieldRemark).Elem().Interface()
								} else {
									newValue = "<nil>"
								}
								log.Info(fmt.Sprintf("[PATCH %s] field: %q: %v --> %v", fieldRemark, typ.Name(), oldValue, newValue))
							} else {
								log.Info(fmt.Sprintf("[PATCH %s] field: %q: %v --> %v", fieldRemark, typ.Name(),
									oldVal.FieldByName(fieldRemark).Interface(), newVal.FieldByName(fieldRemark).Interface()))
							}
							oldVal.FieldByName(fieldRemark).Set(newVal.FieldByName(fieldRemark)) // set old value by new value
						}
					}
				}
				// Make sure the type of "Order" is pointer to golang base type.
				fieldOrder := "Order"
				if oldVal.FieldByName(fieldOrder).CanSet() {
					if newVal.FieldByName(fieldOrder).IsValid() { // WARN: oldVal.FieldByName(fieldOrder) maybe <invalid reflect.Value>
						if !newVal.FieldByName(fieldOrder).IsZero() {
							// output log must before set value.
							if newVal.FieldByName(fieldOrder).Kind() == reflect.Pointer {
								var oldValue, newValue any
								if !oldVal.FieldByName(fieldOrder).IsNil() {
									oldValue = oldVal.FieldByName(fieldOrder).Elem().Interface()
								} else {
									oldValue = "<nil>"
								}
								if !newVal.FieldByName(fieldOrder).IsNil() {
									newValue = newVal.FieldByName(fieldOrder).Elem().Interface()
								} else {
									newValue = "<nil>"
								}
								log.Info(fmt.Sprintf("[PATCH %s] field: %q: %v --> %v", fieldOrder, typ.Name(), oldValue, newValue))
							} else {
								log.Info(fmt.Sprintf("[PATCH %s] field: %q: %v --> %v", fieldOrder, typ.Name(),
									oldVal.FieldByName(fieldOrder).Interface(), newVal.FieldByName(fieldOrder).Interface()))
							}
							oldVal.FieldByName(fieldOrder).Set(newVal.FieldByName(fieldOrder)) // set old value by new value.
						}
					}
				}
				continue

			default:
				continue
			}
		}
		if !oldVal.Field(i).CanSet() {
			log.Warnf("field %q is cannot set, skip", typ.Field(i).Name)
			continue
		}
		if !newVal.Field(i).IsValid() {
			// log.Warnf("field %s is invalid, skip", typ.Field(i).Name)
			continue
		}
		// base type such like int and string have default value(zero value).
		// If the struct field(the field type is golang base type) supported by patch update,
		// the field type must be pointer to base type, such like *string, *int.
		if newVal.Field(i).IsZero() {
			// log.Warnf("field %s is zero value, skip", typ.Field(i).Name)
			// log.Warnf("DeepEqual: %v : %v : %v : %v", typ.Field(i).Name, newVal.Field(i).Interface(), oldVal.Field(i).Interface(), reflect.DeepEqual(newVal.Field(i), oldVal.Field(i)))
			continue
		}
		// output log must before set value.
		if newVal.Field(i).Kind() == reflect.Pointer {
			var oldValue, newValue any
			if !oldVal.Field(i).IsNil() {
				oldValue = oldVal.Field(i).Elem().Interface()
			} else {
				oldValue = "<nil>"
			}
			if !newVal.Field(i).IsNil() {
				newValue = newVal.Field(i).Elem().Interface()
			} else {
				newValue = "<nil>"
			}
			log.Info(fmt.Sprintf("[PATCH %s] field: %q: %v --> %v", typ.Name(), typ.Field(i).Name, oldValue, newValue))
		} else {
			log.Info(fmt.Sprintf("[PATCH %s] field: %q: %v --> %v", typ.Name(), typ.Field(i).Name, oldVal.Field(i).Interface(), newVal.Field(i).Interface()))
		}
		oldVal.Field(i).Set(newVal.Field(i)) // set old value by new value
	}
}

// getCallerInfo returns the file name and line number of the caller
func getCallerInfo(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}
	return filepath.Base(file), line
}

func extractConfig[M types.Model](cfg ...*types.ControllerConfig[M]) (handler func(ctx *types.DatabaseContext) types.Database[M], db any) {
	if len(cfg) > 0 {
		if cfg[0] != nil {
			db = cfg[0].DB
		}
	}
	handler = func(ctx *types.DatabaseContext) types.Database[M] {
		fn := database.Database[M](ctx)
		if len(cfg) > 0 {
			if cfg[0] != nil {
				if len(cfg[0].TableName) > 0 {
					fn = database.Database[M](ctx).WithDB(cfg[0].DB).WithTable(cfg[0].TableName)
				} else {
					fn = database.Database[M](ctx).WithDB(cfg[0].DB)
				}
			}
		}
		return fn
	}
	return handler, db
}

// startControllerSpan starts a span for controller operations
func startControllerSpan[M types.Model](c *gin.Context, phase consts.Phase) (context.Context, trace.Span) {
	// Get the model name(struct name).
	modelName := reflect.TypeOf(*new(M)).Elem().Name()

	// Create child span for controller operation
	spanName := fmt.Sprintf("Controller.%s %s", phase.MethodName(), modelName)
	spanCtx, span := otel.StartSpan(c.Request.Context(), spanName)

	// Update request context with new span context
	c.Request = c.Request.WithContext(spanCtx)

	// Add controller-specific attributes
	otel.AddSpanTags(span, map[string]any{
		"component":            "controller",
		"controller.operation": phase.MethodName(),
		"controller.model":     modelName,
		"controller.method":    c.Request.Method,
		"controller.path":      c.FullPath(),
	})

	return spanCtx, span
}

// traceServiceHook traces the service hook execution.
func traceServiceHook[M types.Model](parentCtx context.Context, phase consts.Phase, fn func(context.Context) error) error {
	// Get the model name(struct name).
	modelName := reflect.TypeOf(*new(M)).Elem().Name()

	// Create children span for service operation
	spanName := fmt.Sprintf("Service.%s %s", phase.MethodName(), modelName)
	spanCtx, span := otel.StartSpan(parentCtx, spanName)
	defer span.End()

	// // Update request context
	// c.Request = c.Request.WithContext(spanCtx)

	// // Get caller information
	// file, line := getCallerInfo(2)

	// Add service-specific attributes
	otel.AddSpanTags(span, map[string]any{
		"component":         "service",
		"service.operation": phase.MethodName(),
		"service.model":     modelName,
		// "code.file":         file,
		// "code.line":         line,
	})

	// Declare error variable for use in defer
	var err error

	// Record start time and ensure duration + success recorded at the end
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		otel.AddSpanTags(span, map[string]any{
			"hook.duration_ms": duration.Milliseconds(),
			"hook.success":     err == nil,
		})
		if err != nil {
			otel.RecordError(span, err)
		}
	}()

	err = fn(spanCtx)
	return err
}

// traceServiceOperation traces the service operation.
func traceServiceOperation[M types.Model, RSP types.Response](parentCtx context.Context, phase consts.Phase, fn func(context.Context) (RSP, error)) (RSP, error) {
	// Get the model name(struct name).
	modelName := reflect.TypeOf(*new(M)).Elem().Name()

	// Create children span for service operation
	spanName := fmt.Sprintf("Service.%s %s", phase.MethodName(), modelName)
	spanCtx, span := otel.StartSpan(parentCtx, spanName)
	defer span.End()

	// // Update request context
	// c.Request = c.Request.WithContext(spanCtx)

	// // Get caller information
	// file, line := getCallerInfo(2)

	// Add service-specific attributes
	otel.AddSpanTags(span, map[string]any{
		"component":         "service",
		"service.operation": phase.MethodName(),
		"service.model":     modelName,
		// "code.file":         file,
		// "code.line":         line,
	})

	// Declare error variable for use in defer
	var err error
	var rsp RSP

	// Record start time and ensure duration + success recorded at the end
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		otel.AddSpanTags(span, map[string]any{
			"hook.duration_ms": duration.Milliseconds(),
			"hook.success":     err == nil,
		})
		if err != nil {
			otel.RecordError(span, err)
		}
	}()

	rsp, err = fn(spanCtx)
	return rsp, err
}

// traceServiceExport traces the service export operation.
func traceServiceExport[M types.Model, T []byte](parentCtx context.Context, phase consts.Phase, fn func(context.Context) (T, error)) (T, error) {
	// Get the model name(struct name).
	modelName := reflect.TypeOf(*new(M)).Elem().Name()

	// Create children span for service operation
	spanName := fmt.Sprintf("Service.%s %s", phase.MethodName(), modelName)
	spanCtx, span := otel.StartSpan(parentCtx, spanName)
	defer span.End()

	// // Update request context
	// c.Request = c.Request.WithContext(spanCtx)

	// // Get caller information
	// file, line := getCallerInfo(2)

	// Add service-specific attributes
	otel.AddSpanTags(span, map[string]any{
		"component":         "service",
		"service.operation": phase.MethodName(),
		"service.model":     modelName,
		// "code.file":         file,
		// "code.line":         line,
	})

	// Declare error variable for use in defer
	var err error
	var data T

	// Record start time and ensure duration + success recorded at the end
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		otel.AddSpanTags(span, map[string]any{
			"hook.duration_ms": duration.Milliseconds(),
			"hook.success":     err == nil,
		})
		if err != nil {
			otel.RecordError(span, err)
		}
	}()

	data, err = fn(spanCtx)
	return data, err
}

// traceServiceImport traces the service import operation.
func traceServiceImport[M types.Model](parentCtx context.Context, phase consts.Phase, fn func(context.Context) ([]M, error)) ([]M, error) {
	// Get the model name(struct name).
	modelName := reflect.TypeOf(*new(M)).Elem().Name()

	// Create children span for service operation
	spanName := fmt.Sprintf("Service.%s %s", phase.MethodName(), modelName)
	spanCtx, span := otel.StartSpan(parentCtx, spanName)
	defer span.End()

	// // Update request context
	// c.Request = c.Request.WithContext(spanCtx)

	// // Get caller information
	// file, line := getCallerInfo(2)

	// Add service-specific attributes
	otel.AddSpanTags(span, map[string]any{
		"component":         "service",
		"service.operation": phase.MethodName(),
		"service.model":     modelName,
		// "code.file":         file,
		// "code.line":         line,
	})

	// Declare error variable for use in defer
	var err error
	var ml []M

	// Record start time and ensure duration + success recorded at the end
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		otel.AddSpanTags(span, map[string]any{
			"hook.duration_ms": duration.Milliseconds(),
			"hook.success":     err == nil,
		})
		if err != nil {
			otel.RecordError(span, err)
		}
	}()

	ml, err = fn(spanCtx)
	return ml, err
}

// handleServiceError handles ServiceError
func handleServiceError(c *gin.Context, ctx *types.ServiceContext, err error) {
	// Check if it's a ServiceError
	var serviceErr *types.ServiceError
	if errors.As(err, &serviceErr) {
		ResponseJSON(c, CodeFailure.WithStatus(serviceErr.StatusCode).WithErr(err))
		return
	}

	// Default error handling
	ResponseJSON(c, CodeFailure.WithErr(err))
}

// logRequest logs the HTTP request using zap logger if enabled in config
func logRequest(log types.Logger, phase consts.Phase, req any) {
	if !config.App.Logger.Controller.LogRequest {
		return
	}
	log.Infow("request", zap.String("phase", phase.MethodName()), zap.Any("request", req))
}

// logResponse logs the HTTP response using zap logger if enabled in config
func logResponse(log types.Logger, phase consts.Phase, rsp any) {
	if !config.App.Logger.Controller.LogResponse {
		return
	}
	log.Infow("response", zap.String("phase", phase.MethodName()), zap.Any("response", rsp))
}
