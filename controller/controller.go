package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/ds/queue/circularbuffer"
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	"github.com/forbearing/gst/logger"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/pkg/auditmanager"
	"github.com/forbearing/gst/pkg/filetype"
	"github.com/forbearing/gst/provider/otel"
	. "github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	pluralize "github.com/gertd/go-pluralize"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

// TODO: 记录失败的操作.

/*
1.Model 层处理单个 types.Model, 功能: 数据预处理
2.Service 层处理多个 types.Model, 功能: 具体的业务逻辑
3.Database 层处理多个 types.Model, 功能: 数据库的增删改查,redis缓存等.
4.这三层都能对源数据进行修改, 因为:
  - Model 的实现对象必须是结构体指针
  - types.Service[M types.Model]: types.Service 泛型接口的类型约束是 types.Model 接口
  - types.Database[M types.Model]: types.Database 泛型接口的类型约束就是 types.Model 接口
  以上这三个条件自己慢慢体会吧.
5.用户自定义的 model:
  必须继承 model.Base 结构体, 因为这个结构体实现了 types.Model 接口
  用户只需要添加自己的字段和相应的 tag 和方法即可.
  如果想要给 types.Model 在数据库中创建对象的表, 请在 init() 中调用 register 函数注册一下即可, 比如 register[*Asset]()
  如果需要在创建表格的同时创建记录, 也可以通过 register 函数来做, 比如 register[*Asset](asset01, asset02, asset03)
  这里的 asset01, asset02, asset03 的类型是 *model.Asset.
6.用户自定义 service
  必须继承 service.base 结构体, 因为这个结构体实现了 types.Service[types.Model] 接口
  用户只需要覆盖默认的方法就行了
如果有额外的业务逻辑, 在 init() 中调用 register 函数注册一下自己定义的 service, 例如: register[*asset, *model.Asset](new(asset))
如果 service.Asset 有自定义字段, 可以这样: register[*asset, *model.Asset](&asset{SheetName: "资产类别清单"})

处理资源顺序:
    通用流程: Request -> ServiceBefore -> ModelBefore -> Database -> ModelAfter -> ServiceAfter -> Response.
	导入数据: Request -> ServiceBefore -> Import -> ModelBefore ->  Database -> ModelAfter -> ServiceAfter -> Response.
	导出数据: Request -> ServiceBefore -> ModelBefore -> Database -> ModelAfter -> ServiceAfter -> Export -> Response.

    Import 逻辑类似于 Update 逻辑
	Import 的 Model 的 UpdateBefore() 在 service 层里面处理, ServiceBefore 是可选的
	Export 逻辑类似于 List 逻辑, 只是比 Update 逻辑多了 Export 步骤

其他:
	1.记录操作日志也在 controller 层
*/

const ErrRequestBodyEmpty = "request body is empty"

const defaultLimit = 1000

var (
	pluralizeCli = pluralize.NewClient()

	// Global circular buffer for controller logger
	cb *circularbuffer.CircularBuffer[*modellogmgmt.OperationLog]

	// Global audit manager instance
	am *auditmanager.AuditManager
)

func Init() (err error) {
	// Initialize circular buffer
	if cb, err = circularbuffer.New(int(config.App.Server.CircularBuffer.SizeOperationLog), circularbuffer.WithSafe[*modellogmgmt.OperationLog]()); err != nil {
		return err
	}

	// Initialize audit manager
	am = auditmanager.New(&config.App.Audit, cb)

	// Consume operation log.
	go am.Consume()

	return nil
}

func Clean() {
	operationLogs := make([]*modellogmgmt.OperationLog, 0, config.App.Server.CircularBuffer.SizeOperationLog)
	for !cb.IsEmpty() {
		ol, _ := cb.Dequeue()
		operationLogs = append(operationLogs, ol)
	}
	if len(operationLogs) > 0 {
		if err := database.Database[*modellogmgmt.OperationLog](nil).WithLimit(-1).WithBatchSize(100).Create(operationLogs...); err != nil {
			zap.S().Error(err)
		}
	}
}

// Create is a generic function to product gin handler to create one resource.
// The resource type depends on the type of interface types.Model.
func Create[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	CreateFactory[M, REQ, RSP]()(c)
}

// CreateFactory is a factory function that produces a gin handler for creating one resource.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles resource creation in database
//   - Service hooks (CreateBefore/CreateAfter) are executed for business logic
//   - Processing flow: Request -> ServiceBefore -> ModelBefore -> Database -> ModelAfter -> ServiceAfter -> Response
//   - The request body is directly bound to the model type M
//   - Automatic setting of CreatedBy/UpdatedBy fields from context
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over resource creation
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.Create -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic and database operations
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP POST requests
//
// HTTP Response:
//   - Success: 201 Created with the created resource data
//   - Error: 400 Bad Request for invalid parameters, 500 Internal Server Error for other failures
//
// Examples:
//
// Unified types (automatic mode):
//
//	CreateFactory[*model.User, *model.User, *model.User]()
//
// Custom types (manual mode):
//
//	CreateFactory[*model.User, *CreateUserRequest, *CreateUserResponse]()
func CreateFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var err error
		var reqErr error

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_CREATE)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_CREATE)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_CREATE)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_CREATE, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_CREATE, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_CREATE)
				return svc.Create(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_CREATE, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		typ := reflect.TypeOf(*new(M)).Elem()
		req := reflect.New(typ).Interface().(M) //nolint:errcheck
		if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
			log.Error(reqErr)
			ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
			otel.RecordError(span, err)
			return
		}
		if errors.Is(reqErr, io.EOF) {
			log.Warn(ErrRequestBodyEmpty)
		} else {
			req.SetCreatedBy(c.GetString(consts.CTX_USERNAME))
			req.SetUpdatedBy(c.GetString(consts.CTX_USERNAME))
			log.Infoz("create", zap.Object(reflect.TypeOf(*new(M)).Elem().String(), req))
		}
		logRequest(log, consts.PHASE_CREATE, req)

		// 1.Perform business logic processing before create resource.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_CREATE_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_CREATE_BEFORE)
			return svc.CreateBefore(serviceCtxBefore, req)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		// 2.Create resource in database.
		// database.Database().Delete just set "deleted_at" field to current time, not really delete.
		// We should update it instead of creating it, and update the "created_at" and "updated_at" field.
		// NOTE: WithExpand(req.Expands()...) is not a good choices.
		// if err := database.Database[M]().WithExpand(req.Expands()...).Update(req); err != nil {
		if !errors.Is(reqErr, io.EOF) {
			if err = handler(types.NewDatabaseContext(c)).WithExpand(req.Expands()).Create(req); err != nil {
				log.Error(err)
				ResponseJSON(c, CodeFailure.WithErr(err))
				otel.RecordError(span, err)
				return
			}
		}
		// 3.Perform business logic processing after create resource
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_CREATE_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_CREATE_AFTER)
			return svc.CreateAfter(serviceCtxAfter, req)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		record, _ := json.Marshal(req)
		reqData, _ := json.Marshal(req)
		respData, _ := json.Marshal(req)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_CREATE,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	RecordID:  req.GetID(),
		// 	Record:    util.BytesToString(record),
		// 	Request:   util.BytesToString(reqData),
		// 	Response:  util.BytesToString(respData),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		if err = am.RecordOperation(types.NewDatabaseContext(c), req, &modellogmgmt.OperationLog{
			OP:        consts.OP_CREATE,
			Model:     typ.Name(),
			RecordID:  req.GetID(),
			Record:    util.BytesToString(record),
			Request:   util.BytesToString(reqData),
			Response:  util.BytesToString(respData),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		logResponse(log, consts.PHASE_CREATE, req)
		ResponseJSON(c, CodeSuccess.WithStatus(http.StatusCreated), req)
	}
}

// Delete is a generic function to product gin handler to delete one or multiple resources.
// The resource type depends on the type of interface types.Model.
//
// Resource id must be specify and all resources that id matched will be deleted in database.
//
// Delete one resource:
// - specify resource `id` in "router parameter", eg: localhost:9000/api/myresource/myid
// - specify resource `id` in "query parameter", eg: localhost:9000/api/myresource?id=myid
//
// Delete multiple resources:
// - specify resource `id` slice in "http body data".
func Delete[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	DeleteFactory[M, REQ, RSP]()(c)
}

// DeleteFactory is a factory function that produces a gin handler for deleting one or multiple resources.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles resource deletion from database
//   - Service hooks (DeleteBefore/DeleteAfter) are executed for business logic
//   - Processing flow: Request -> ServiceBefore -> ModelBefore -> Database -> ModelAfter -> ServiceAfter -> Response
//   - Supports multiple deletion methods:
//   - Route parameter: DELETE /api/users/123
//   - Query parameter: DELETE /api/users?id=123
//   - Request body: DELETE /api/users with JSON array ["id1", "id2", "id3"]
//   - Automatic deduplication of resource IDs
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over resource deletion logic
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.Delete -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic and database operations
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP DELETE requests
//
// HTTP Response:
//   - Success: 204 No Content (unified types) or 200 OK (custom types) with optional response data
//   - Error: 400 Bad Request for invalid parameters, 404 Not Found if resource doesn't exist, 500 Internal Server Error for other failures
//
// Examples:
//
// Unified types (automatic mode):
//
//	DeleteFactory[*model.User, *model.User, *model.User]()
//	// Supports: DELETE /users/123, DELETE /users?id=123, DELETE /users with ["id1", "id2"]
//
// Custom types (manual mode):
//
//	DeleteFactory[*model.User, *DeleteUserRequest, *DeleteUserResponse]()
//	// Service layer controls all deletion logic
func DeleteFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_DELETE)
		defer span.End()

		cctx := types.NewControllerContext(c)
		log := logger.Controller.WithControllerContext(cctx, consts.PHASE_DELETE)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_DELETE)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var err error
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr := c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			logRequest(log, consts.PHASE_DELETE, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_DELETE, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_DELETE)
				return svc.Delete(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_DELETE, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		// The underlying type of interface types.Model must be pointer to structure, such as *model.User.
		// 'typ' is the structure type, such as: model.User.
		typ := reflect.TypeOf(*new(M)).Elem()
		ml := make([]M, 0)
		idsSet := make(map[string]struct{})

		addID := func(id string) {
			if len(id) == 0 {
				return
			}
			if _, exists := idsSet[id]; exists {
				return
			}
			// 'm' is the structure value such as: &model.User{ID: myid, Name: myname}.
			m := reflect.New(typ).Interface().(M) //nolint:errcheck
			m.SetID(id)
			ml = append(ml, m)
			idsSet[id] = struct{}{}
		}

		// Delete one record accoding to "query parameter `id`".
		if id, ok := c.GetQuery(consts.QUERY_ID); ok {
			addID(id)
		}
		// Delete one record accoding to "route parameter `id`".
		if len(cfg) > 0 {
			addID(cctx.Params[util.Deref(cfg[0]).ParamName])
		}
		// Delete multiple records accoding to "http body data".
		bodyIds := make([]string, 0)
		if err := c.ShouldBindJSON(&bodyIds); err == nil && len(bodyIds) > 0 {
			for _, id := range bodyIds {
				addID(id)
			}
		}

		ids := make([]string, 0, len(idsSet))
		for id := range idsSet {
			ids = append(ids, id)
		}
		log.Info(fmt.Sprintf("%s delete %v", typ.Name(), ids))

		// 1.Perform business logic processing before delete resources.
		// TODO: Should there be one service hook(DeleteBefore), or multiple?
		for _, m := range ml {
			var serviceCtxBefore *types.ServiceContext
			if err := traceServiceHook[M](ctrlSpanCtx, consts.PHASE_DELETE_BEFORE, func(spanCtx context.Context) error {
				serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_DELETE_BEFORE)
				return svc.DeleteBefore(serviceCtxBefore, m)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtxBefore, err)
				otel.RecordError(span, err)
				return
			}
		}

		// find out the records and record to operation log.
		copied := make([]M, len(ml))
		for i := range ml {
			m := reflect.New(typ).Interface().(M) //nolint:errcheck
			m.SetID(ml[i].GetID())
			if err := handler(types.NewDatabaseContext(c)).WithExpand(m.Expands()).Get(m, ml[i].GetID()); err != nil {
				log.Error(err)
				otel.RecordError(span, err)
			}
			copied[i] = m
		}

		// 2.Delete resources in database.
		if err := handler(types.NewDatabaseContext(c)).Delete(ml...); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// 3.Perform business logic processing after delete resources.
		// TODO: Should there be one service hook(DeleteAfter), or multiple?
		for _, m := range ml {
			var serviceCtxAfter *types.ServiceContext
			if err := traceServiceHook[M](ctrlSpanCtx, consts.PHASE_DELETE_AFTER, func(spanCtx context.Context) error {
				serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_DELETE_AFTER)
				return svc.DeleteAfter(serviceCtxAfter, m)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtxAfter, err)
				otel.RecordError(span, err)
				return
			}
		}

		// 4.record operation log to database.
		for i := range ml {
			record, _ := json.Marshal(copied[i])
			// cb.Enqueue(&modellogmgmt.OperationLog{
			// 	OP:        consts.OP_DELETE,
			// 	Model:     typ.Name(),
			// 	Table:     tableName,
			// 	RecordID:  ml[i].GetID(),
			// 	Record:    util.BytesToString(record),
			// 	IP:        c.ClientIP(),
			// 	User:      c.GetString(consts.CTX_USERNAME),
			// 	RequestID: c.GetString(consts.REQUEST_ID),
			// 	URI:       c.Request.RequestURI,
			// 	Method:    c.Request.Method,
			// 	UserAgent: c.Request.UserAgent(),
			// })
			m := reflect.New(typ).Interface().(M) //nolint:errcheck
			if err := am.RecordOperation(types.NewDatabaseContext(c), m, &modellogmgmt.OperationLog{
				OP:        consts.OP_DELETE,
				Model:     typ.Name(),
				RecordID:  ml[i].GetID(),
				Record:    util.BytesToString(record),
				IP:        c.ClientIP(),
				User:      c.GetString(consts.CTX_USERNAME),
				RequestID: c.GetString(consts.REQUEST_ID),
				URI:       c.Request.RequestURI,
				Method:    c.Request.Method,
				UserAgent: c.Request.UserAgent(),
			}); err != nil {
				log.Warn(err)
			}
		}

		ResponseJSON(c, CodeSuccess.WithStatus(http.StatusNoContent))
	}
}

// Update is a generic function to product gin handler to update one resource.
// The resource type depends on the type of interface types.Model.
//
// Update will update one resource and resource "ID" must be specified,
// which can be specify in "router parameter `id`" or "http body data".
//
// "router parameter `id`" has more priority than "http body data".
// It will skip decode id from "http body data" if "router parameter `id`" not empty.
func Update[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	UpdateFactory[M, REQ, RSP]()(c)
}

// UpdateFactory is a factory function that produces a gin handler for updating one resource.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles resource update in database
//   - Service hooks (UpdateBefore/UpdateAfter) are executed for business logic
//   - Processing flow: Request -> ServiceBefore -> ModelBefore -> Database -> ModelAfter -> ServiceAfter -> Response
//   - Resource ID can be specified in two ways (route parameter has higher priority):
//   - Route parameter: PUT /api/users/123
//   - Request body: PUT /api/users with JSON {"id": "123", "name": "new name"}
//   - Automatic preservation of original created_at and created_by fields
//   - Automatic setting of updated_by field to current user
//   - Validates resource existence before update
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over resource update logic
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.Update -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic, validation, and database operations
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP PUT requests
//
// HTTP Response:
//   - Success: 200 OK with the updated resource data
//   - Error: 400 Bad Request for invalid parameters, 404 Not Found if resource doesn't exist, 500 Internal Server Error for other failures
//
// Resource ID Priority:
//  1. Route parameter (/api/users/123) - highest priority
//  2. Request body ID field - fallback if route parameter is empty
//  3. Error if neither is provided
//
// Examples:
//
// Unified types (automatic mode):
//
//	UpdateFactory[*model.User, *model.User, *model.User]()
//	// Supports: PUT /users/123 or PUT /users with {"id": "123", "name": "John"}
//
// Custom types (manual mode):
//
//	UpdateFactory[*model.User, *UpdateUserRequest, *UpdateUserResponse]()
//	// Service layer controls all update logic
func UpdateFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var err error
		var reqErr error

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_UPDATE)
		defer span.End()

		cctx := types.NewControllerContext(c)
		log := logger.Controller.WithControllerContext(cctx, consts.PHASE_UPDATE)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_UPDATE)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_UPDATE, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_UPDATE, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_UPDATE)
				return svc.Update(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_UPDATE, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		typ := reflect.TypeOf(*new(M)).Elem()
		req := reflect.New(typ).Interface().(M) //nolint:errcheck
		if reqErr := c.ShouldBindJSON(&req); reqErr != nil {
			log.Error(reqErr)
			ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
			otel.RecordError(span, err)
			return
		}
		logRequest(log, consts.PHASE_UPDATE, req)

		// param id has more priority than http body data id
		var paramID string
		if len(cfg) > 0 {
			paramID = cctx.Params[util.Deref(cfg[0]).ParamName]
		}
		bodyID := req.GetID()
		var id string
		log.Infoz("update from request",
			zap.String("param_id", paramID),
			zap.String("body_id", bodyID),
			zap.Object(reflect.TypeOf(*new(M)).Elem().String(), req),
		)
		if paramID != "" {
			req.SetID(paramID)
			id = paramID
		} else if bodyID != "" {
			paramID = bodyID //nolint:ineffassign,wastedassign
			id = bodyID
		} else {
			log.Error("id missing")
			ResponseJSON(c, CodeFailure.WithErr(errors.New("id missing")))
			otel.RecordError(span, err)
			return
		}

		data := make([]M, 0)
		// The underlying type of interface types.Model must be pointer to structure, such as *model.User.
		// 'typ' is the structure type, such as: model.User.
		// 'm' is the structure value such as: &model.User{ID: myid, Name: myname}.
		m := reflect.New(typ).Interface().(M) //nolint:errcheck
		m.SetID(id)
		// Make sure the record must be already exists.
		if err = handler(types.NewDatabaseContext(c)).WithLimit(1).WithQuery(m).List(&data); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		if len(data) != 1 {
			log.Errorz(fmt.Sprintf("the total number of records query from database not equal to 1(%d)", len(data)), zap.String("id", id))
			ResponseJSON(c, CodeNotFound)
			return
		}

		req.SetCreatedAt(data[0].GetCreatedAt())           // keep original "created_at"
		req.SetCreatedBy(data[0].GetCreatedBy())           // keep original "created_by"
		req.SetUpdatedBy(c.GetString(consts.CTX_USERNAME)) // set updated_by to current user”

		// 1.Perform business logic processing before update resource.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_UPDATE_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_UPDATE_BEFORE)
			return svc.UpdateBefore(serviceCtxBefore, req)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		// 2.Update resource in database.
		log.Infoz("update in database", zap.Object(typ.Name(), req))
		if err = handler(types.NewDatabaseContext(c)).Update(req); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// 3.Perform business logic processing after update resource.
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_UPDATE_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_UPDATE_AFTER)
			return svc.UpdateAfter(serviceCtxAfter, req)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		record, _ := json.Marshal(req)
		reqData, _ := json.Marshal(req)
		respData, _ := json.Marshal(req)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_UPDATE,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	RecordID:  req.GetID(),
		// 	Record:    util.BytesToString(record),
		// 	Request:   util.BytesToString(reqData),
		// 	Response:  util.BytesToString(respData),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		if err = am.RecordOperation(types.NewDatabaseContext(c), req, &modellogmgmt.OperationLog{
			OP:        consts.OP_UPDATE,
			Model:     typ.Name(),
			RecordID:  req.GetID(),
			Record:    util.BytesToString(record),
			Request:   util.BytesToString(reqData),
			Response:  util.BytesToString(respData),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		logResponse(log, consts.PHASE_UPDATE, req)
		ResponseJSON(c, CodeSuccess, req)
	}
}

// Patch is a generic function to product gin handler to partial update one resource.
// The resource type depends on the type of interface types.Model.
//
// resource id must be specified.
// - specified in "query parameter `id`".
// - specified in "router parameter `id`".
//
// which one or multiple resources desired modify.
// - specified in "query parameter".
// - specified in "http body data".
func Patch[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	PatchFactory[M, REQ, RSP]()(c)
}

// PatchFactory is a factory function that produces a gin handler for partially updating one resource.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles partial resource update in database
//   - Service hooks (PatchBefore/PatchAfter) are executed for business logic
//   - Processing flow: Request -> ServiceBefore -> ModelBefore -> Database -> ModelAfter -> ServiceAfter -> Response
//   - Resource ID can be specified in two ways (route parameter has higher priority):
//   - Route parameter: PATCH /api/users/123
//   - Request body: PATCH /api/users with JSON {"id": "123", "name": "new name"}
//   - Only non-zero fields from request are applied to existing resource (partial update)
//   - Automatic setting of updated_by field to current user
//   - Validates resource existence before update
//   - Preserves original created_at and created_by fields automatically
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over partial resource update logic
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.Patch -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic, validation, and database operations
//
// Partial Update Logic (Mode 1):
//   - Retrieves existing resource from database using provided ID
//   - Uses reflection to copy only non-zero fields from request to existing resource
//   - Maintains original timestamps and audit fields
//   - Updates only the modified fields in database
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP PATCH requests
//
// HTTP Response:
//   - Success: 200 OK with the updated resource data
//   - Error: 400 Bad Request for invalid parameters, 404 Not Found if resource doesn't exist, 500 Internal Server Error for other failures
//
// Resource ID Priority:
//  1. Route parameter (/api/users/123) - highest priority
//  2. Request body ID field - fallback if route parameter is empty
//  3. Error if neither is provided
//
// Examples:
//
// Unified types (automatic partial update):
//
//	PatchFactory[*model.User, *model.User, *model.User]()
//	// Request: PATCH /users/123 with {"name": "John"} - only updates name field
//
// Custom types (manual mode):
//
//	PatchFactory[*model.User, *PatchUserRequest, *PatchUserResponse]()
//	// Service layer controls all partial update logic
func PatchFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var id string

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_PATCH)
		defer span.End()

		cctx := types.NewControllerContext(c)
		log := logger.Controller.WithControllerContext(cctx, consts.PHASE_PATCH)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_PATCH)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var err error
			var reqErr error
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_PATCH, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_PATCH, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_PATCH)
				return svc.Patch(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_PATCH, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		typ := reflect.TypeOf(*new(M)).Elem()
		req := reflect.New(typ).Interface().(M) //nolint:errcheck
		if len(cfg) > 0 {
			id = cctx.Params[util.Deref(cfg[0]).ParamName]
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		logRequest(log, consts.PHASE_PATCH, req)
		if len(id) == 0 {
			id = req.GetID()
		}
		if len(id) == 0 {
			log.Error(CodeNotFoundRouteParam)
			ResponseJSON(c, CodeNotFoundRouteParam)
			otel.RecordError(span, errors.New(CodeNotFoundRouteParam.Msg()))
			return
		}
		data := make([]M, 0)
		// The underlying type of interface types.Model must be pointer to structure, such as *model.User.
		// 'typ' is the structure type, such as: model.User.
		// 'm' is the structure value such as: &model.User{ID: myid, Name: myname}.
		m := reflect.New(typ).Interface().(M) //nolint:errcheck
		m.SetID(id)

		// Make sure the record must be already exists.
		if err := handler(types.NewDatabaseContext(c)).WithLimit(1).WithQuery(m).List(&data); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		if len(data) != 1 {
			log.Errorz(fmt.Sprintf("the total number of records query from database not equal to 1(%d)", len(data)), zap.String("id", id))
			ResponseJSON(c, CodeNotFound)
			return
		}
		// req.SetCreatedAt(data[0].GetCreatedAt())
		// req.SetCreatedBy(data[0].GetCreatedBy())
		// req.SetUpdatedBy(c.GetString(CTX_USERNAME))
		data[0].SetUpdatedBy(c.GetString(consts.CTX_USERNAME))

		newVal := reflect.ValueOf(req).Elem()
		oldVal := reflect.ValueOf(data[0]).Elem()
		patchValue(log, typ, oldVal, newVal)
		cur := oldVal.Addr().Interface().(M) //nolint:errcheck

		// 1.Perform business logic processing before partial update resource.
		var serviceCtxBefore *types.ServiceContext
		if err := traceServiceHook[M](ctrlSpanCtx, consts.PHASE_PATCH_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_PATCH_BEFORE)
			return svc.PatchBefore(serviceCtxBefore, cur)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		// 2.Partial update resource in database.
		if err := handler(types.NewDatabaseContext(c)).Update(cur); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// 3.Perform business logic processing after partial update resource.
		var serviceCtxAfter *types.ServiceContext
		if err := traceServiceHook[M](ctrlSpanCtx, consts.PHASE_PATCH_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_PATCH_AFTER)
			return svc.PatchAfter(serviceCtxAfter, cur)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		// NOTE: We should record the `req` instead of `oldVal`, the req is `newVal`.
		record, _ := json.Marshal(req)
		reqData, _ := json.Marshal(req)
		respData, _ := json.Marshal(cur)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_PATCH,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	RecordID:  req.GetID(),
		// 	Record:    util.BytesToString(record),
		// 	Request:   util.BytesToString(reqData),
		// 	Response:  util.BytesToString(respData),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		if err := am.RecordOperation(types.NewDatabaseContext(c), req, &modellogmgmt.OperationLog{
			OP:        consts.OP_PATCH,
			Model:     typ.Name(),
			RecordID:  req.GetID(),
			Record:    util.BytesToString(record),
			Request:   util.BytesToString(reqData),
			Response:  util.BytesToString(respData),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		// NOTE: You should response `oldVal` instead of `req`.
		// The req is `newVal`.
		logResponse(log, consts.PHASE_PATCH, cur)
		ResponseJSON(c, CodeSuccess, cur)
	}
}

// List is a generic function to product gin handler to list resources in backend.
// The resource type deponds on the type of interface types.Model.
//
// If you want make a structure field as query parameter, you should add a "schema"
// tag for it. for example: schema:"name"
//
// TODO:combine query parameter 'page' and 'size' into decoded types.Model
// FIX: retrieve records recursive (current not support in gorm.)
// https://stackoverflow.com/questions/69395891/get-recursive-field-values-in-gorm
// DB.Preload("Category.Category.Category").Find(&Category)
// its works for me.
//
// Query parameters:
//   - All feilds of types.Model's underlying structure but excluding some special fields,
//     such as "password", field value too large, json tag is "-", etc.
//   - `_expand`: strings (multiple items separated by ",").
//     The responsed data to frontend will expanded(retrieve data from external table accoding to foreign key)
//     For examples:
//     /department/myid?_expand=children
//     /department/myid?_expand=children,parent
//   - `_depth`: strings or interger.
//     How depth to retrieve records from datab recursively, default to 1, value scope is [1,99].
//     For examples:
//     /department/myid?_expand=children&_depth=3
//     /department/myid?_expand=children,parent&_depth=10
//   - `_fuzzy`: bool
//     fuzzy match records in database, default to fase.
//     For examples:
//     /department/myid?_fuzzy=true
func List[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	ListFactory[M, REQ, RSP]()(c)
}

// ListFactory is a factory function that produces a gin handler for listing multiple resources.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles resource listing from database
//   - Service hooks (ListBefore/ListAfter) are executed for business logic
//   - Processing flow: QueryParams -> ServiceBefore -> Database -> ServiceAfter -> Response
//   - Supports comprehensive query parameters for filtering, pagination, sorting, and expansion
//   - Automatic caching support with configurable cache control
//   - Built-in pagination with total count calculation
//   - Advanced features: cursor-based pagination, fuzzy search, field expansion, time range filtering
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over resource listing logic
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.List -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic, filtering, and database operations
//
// Supported Query Parameters (Mode 1):
//   - Model fields: All fields with "schema" tag can be used as query parameters
//   - _page: Page number for pagination (starts from 1)
//   - _size: Number of items per page
//   - _expand: Comma-separated list of fields to expand (foreign key relationships)
//   - _depth: Expansion depth for recursive relationships (1-99, default: 1)
//   - _fuzzy: Enable fuzzy matching for string fields (true/false)
//   - _or: Use OR logic instead of AND for multiple conditions (true/false)
//   - _sortby: Field name for sorting (append " desc" for descending order)
//   - _select: Comma-separated list of fields to select
//   - _nocache: Disable caching for this request (true/false)
//   - _nototal: Skip total count calculation for better performance (true/false)
//   - _cursor_value: Cursor value for cursor-based pagination
//   - _cursor_fields: Fields to use for cursor-based pagination
//   - _cursor_next: Direction for cursor pagination (true for next, false for previous)
//   - _start_time: Start time for time range filtering (format: YYYY-MM-DD HH:mm:ss)
//   - _end_time: End time for time range filtering (format: YYYY-MM-DD HH:mm:ss)
//   - _column_name: Column name for time range filtering
//   - _index: Database index hint for query optimization
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP GET requests
//
// HTTP Response (Mode 1):
//   - Success: 200 OK with {"items": [...], "total": count} or cached byte response
//   - Without total: 200 OK with {"items": [...]} when _nototal=true
//   - Error: 400 Bad Request for invalid parameters, 500 Internal Server Error for other failures
//
// HTTP Response (Mode 2):
//   - Success: 200 OK with service-defined response structure
//   - Error: 400 Bad Request for invalid parameters, 500 Internal Server Error for other failures
//
// Examples:
//
// Unified types (automatic listing with rich query features):
//
//	ListFactory[*model.User, *model.User, *model.User]()
//	// GET /users?name=john&_page=1&_size=10&_expand=department&_sortby=created_at desc
//
// Custom types (manual mode):
//
//	ListFactory[*model.User, *ListUsersRequest, *ListUsersResponse]()
//	// Service layer controls all listing logic
//
// Advanced Query Examples:
//
//	// Fuzzy search with pagination
//	GET /users?name=john&_fuzzy=true&_page=1&_size=20
//
//	// Expand relationships with depth
//	GET /departments?_expand=children,parent&_depth=3
//
//	// Cursor-based pagination
//	GET /users?_cursor_value=123&_cursor_next=true&_size=10
//
//	// Time range filtering
//	GET /logs?_start_time=2024-01-01 00:00:00&_end_time=2024-01-31 23:59:59&_column_name=created_at
func ListFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_LIST)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_LIST)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_LIST)
		ctx := types.NewServiceContext(c)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var err error
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if err = c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
				log.Error(err)
				ResponseJSON(c, CodeInvalidParam.WithErr(err))
				otel.RecordError(span, err)
				return
			}
			logRequest(log, consts.PHASE_LIST, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_LIST, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_LIST)
				return svc.List(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_LIST, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		var page, size int
		var startTime, endTime time.Time
		if pageStr, ok := c.GetQuery(consts.QUERY_PAGE); ok {
			page, _ = strconv.Atoi(pageStr)
		}
		if sizeStr, ok := c.GetQuery(consts.QUERY_SIZE); ok {
			size, _ = strconv.Atoi(sizeStr)
		}
		columnName, _ := c.GetQuery(consts.QUERY_COLUMN_NAME)
		index, _ := c.GetQuery(consts.QUERY_INDEX)
		selects, _ := c.GetQuery(consts.QUERY_SELECT)
		if startTimeStr, ok := c.GetQuery(consts.QUERY_START_TIME); ok {
			startTime, _ = time.ParseInLocation(consts.DATE_TIME_LAYOUT, startTimeStr, time.Local)
		}
		if endTimeStr, ok := c.GetQuery(consts.QUERY_END_TIME); ok {
			endTime, _ = time.ParseInLocation(consts.DATE_TIME_LAYOUT, endTimeStr, time.Local)
		}

		// The underlying type of interface types.Model must be pointer to structure, such as *model.User.
		// 'typ' is the structure type, such as: model.User.
		// 'm' is the structure value, such as: &model.User{ID: myid, Name: myname}.
		typ := reflect.TypeOf(*new(M)).Elem() // the real underlying structure type
		m := reflect.New(typ).Interface().(M) //nolint:errcheck

		// FIXME: failed to convert value when size value is -1.
		if err := schema.NewDecoder().Decode(m, c.Request.URL.Query()); err != nil {
			log.Warn(fmt.Sprintf("failed to decode uri query parameter into model: %s", err))
		}
		log.Infoz(fmt.Sprintf("%s: list query parameter", typ.Name()), zap.Object(typ.String(), m))

		var err error
		var or bool
		var fuzzy bool
		var expands []string
		var cursorNext bool
		var nototal bool // default enable total.
		cursorValue := c.Query(consts.QUERY_CURSOR_VALUE)
		cursorFields := c.Query(consts.QUERY_CURSOR_FIELDS)
		nocache := true // default disable cache.
		depth := 1
		data := make([]M, 0)
		if nocacheStr, ok := c.GetQuery(consts.QUERY_NOCACHE); ok {
			var _nocache bool
			if _nocache, err = strconv.ParseBool(nocacheStr); err == nil {
				nocache = _nocache
			}
		}
		if orStr, ok := c.GetQuery(consts.QUERY_OR); ok {
			or, _ = strconv.ParseBool(orStr)
		}
		if fuzzyStr, ok := c.GetQuery(consts.QUERY_FUZZY); ok {
			fuzzy, _ = strconv.ParseBool(fuzzyStr)
		}
		if cursorNextStr, ok := c.GetQuery(consts.QUERY_CURSOR_NEXT); ok {
			cursorNext, _ = strconv.ParseBool(cursorNextStr)
		}
		if depthStr, ok := c.GetQuery(consts.QUERY_DEPTH); ok {
			depth, _ = strconv.Atoi(depthStr)
			if depth < 1 || depth > 99 {
				depth = 1
			}
		}
		if expandStr, ok := c.GetQuery(consts.QUERY_EXPAND); ok {
			var _expands []string
			items := strings.Split(expandStr, ",")
			if len(items) > 0 {
				if items[0] == consts.VALUE_ALL { // expand all feilds
					items = m.Expands()
				}
			}
			for _, e := range m.Expands() {
				for _, item := range items {
					if strings.EqualFold(item, e) {
						_expands = append(_expands, e)
					}
				}
			}
			// fmt.Println("_expends: ", _expands)
			fieldsMap := make(map[string]reflect.Kind)
			for i := range typ.NumField() {
				fieldsMap[typ.Field(i).Name] = typ.Field(i).Type.Kind()
			}
			for _, e := range _expands {
				// If the expanding field not exists in the structure fiedls, skip depth expand.
				kind, found := fieldsMap[e]
				if !found {
					expands = append(expands, e)
					continue
				}
				// If the expanding field exists in the structure but the kind is not slice, skip depth expand.
				if kind != reflect.Slice {
					expands = append(expands, e)
					continue
				}
				t := make([]string, depth)
				for i := range depth {
					t[i] = e
				}
				// fmt.Println("t: ", t)
				// If expand="Children" and depth=3, the depth expanded is "Children.Children.Children"
				expands = append(expands, strings.Join(t, "."))
			}
			// fmt.Println("expands: ", expands)
		}

		// 1.Perform business logic processing before list resources.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_LIST_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_LIST_BEFORE)
			return svc.ListBefore(serviceCtxBefore, &data)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		sortBy, _ := c.GetQuery(consts.QUERY_SORTBY)
		// 2.List resources from database.
		if size == 0 {
			size = defaultLimit
		}
		if err = handler(types.NewDatabaseContext(c)).
			WithPagination(page, size).
			WithIndex(index).
			WithSelect(strings.Split(selects, ",")...).
			WithQuery(svc.Filter(ctx, m), types.QueryConfig{
				FuzzyMatch: fuzzy,
				AllowEmpty: true,
				UseOr:      or,
				RawQuery:   svc.FilterRaw(ctx),
			}).
			WithCursor(cursorValue, cursorNext, cursorFields).
			WithExclude(m.Excludes()).
			WithExpand(expands, sortBy).
			WithOrder(sortBy).
			WithTimeRange(columnName, startTime, endTime).
			WithCache(!nocache).
			List(&data); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// 3.Perform business logic processing after list resources.
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_LIST_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_LIST_AFTER)
			return svc.ListAfter(serviceCtxAfter, &data)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}
		total := new(int64)
		nototalStr, _ := c.GetQuery(consts.QUERY_NOTOTAL)
		nototal, _ = strconv.ParseBool(nototalStr)
		// NOTE: Total count is not provided when using cursor-based pagination.
		if !nototal && len(cursorValue) == 0 {
			if err = handler(types.NewDatabaseContext(c)).
				// WithPagination(page, size). // NOTE: WithPagination should not apply in Count method.
				// WithSelect(strings.Split(selects, ",")...). // NOTE: WithSelect should not apply in Count method.
				WithIndex(index).
				WithQuery(svc.Filter(ctx, m), types.QueryConfig{
					FuzzyMatch: fuzzy,
					AllowEmpty: true,
					UseOr:      or,
					RawQuery:   svc.FilterRaw(ctx),
				}).
				WithExclude(m.Excludes()).
				WithTimeRange(columnName, startTime, endTime).
				WithCache(!nocache).
				Count(total); err != nil {
				log.Error(err)
				ResponseJSON(c, CodeFailure.WithErr(err))
				otel.RecordError(span, err)
				return
			}
		}

		// 4.record operation log to database.
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_LIST,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		if err = am.RecordOperation(types.NewDatabaseContext(c), m, &modellogmgmt.OperationLog{
			OP:        consts.OP_LIST,
			Model:     typ.Name(),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		log.Infoz(fmt.Sprintf("%s: length: %d, total: %d", typ.Name(), len(data), *total), zap.Object(typ.Name(), m))
		if !nototal {
			ResponseJSON(c, CodeSuccess, gin.H{
				"items": data,
				"total": *total,
			})
		} else {
			ResponseJSON(c, CodeSuccess, gin.H{
				"items": data,
			})
		}
	}
}

// Get is a generic function to product gin handler to list resource in backend.
// The resource type deponds on the type of interface types.Model.
//
// Query parameters:
//   - `_expand`: strings (multiple items separated by ",").
//     The responsed data to frontend will expanded(retrieve data from external table accoding to foreign key)
//     For examples:
//     /department/myid?_expand=children
//     /department/myid?_expand=children,parent
//   - `_depth`: strings or interger.
//     How depth to retrieve records from datab recursively, default to 1, value scope is [1,99].
//     For examples:
//     /department/myid?_expand=children&_depth=3
//     /department/myid?_expand=children,parent&_depth=10
//
// Route parameters:
// - id: string or integer.
func Get[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	GetFactory[M, REQ, RSP]()(c)
}

// GetFactory is a factory function that produces a gin handler for retrieving a single resource.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles resource retrieval from database
//   - Service hooks (GetBefore/GetAfter) are executed for business logic
//   - Processing flow: RouteParam -> ServiceBefore -> Database -> ServiceAfter -> Response
//   - Supports comprehensive query parameters for field expansion, caching, and selection
//   - Automatic caching support with configurable cache control
//   - Advanced features: deep recursive expansion, field selection, cache bypass
//   - Resource ID must be provided via route parameter (e.g., /api/users/123)
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over resource retrieval logic
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.Get -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic and database operations
//
// Supported Query Parameters (Mode 1):
//   - _expand: Comma-separated list of fields to expand (foreign key relationships)
//     Special value "all" expands all available fields defined in model's Expands() method
//   - _depth: Expansion depth for recursive relationships (1-99, default: 1)
//     Controls how many levels deep to expand nested relationships
//   - _select: Comma-separated list of fields to select from database
//   - _nocache: Disable caching for this request (true/false, default: true - cache disabled)
//   - _index: Database index hint for query optimization
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP GET requests
//
// HTTP Response (Mode 1):
//   - Success: 200 OK with resource data or cached byte response
//   - Not Found: 404 Not Found when resource doesn't exist or has empty ID/CreatedAt
//   - Error: 400 Bad Request for missing route parameter, 500 Internal Server Error for other failures
//
// HTTP Response (Mode 2):
//   - Success: 200 OK with service-defined response structure
//   - Error: 400 Bad Request for invalid parameters, 500 Internal Server Error for other failures
//
// Examples:
//
// Unified types (automatic retrieval with expansion and caching):
//
//	GetFactory[*model.User, *model.User, *model.User]()
//	// GET /users/123?_expand=department,roles&_depth=2&_nocache=false
//
// Custom types (manual mode):
//
//	GetFactory[*model.User, *GetUserRequest, *GetUserResponse]()
//	// Service layer controls all retrieval logic
//
// Advanced Query Examples:
//
//	// Expand all relationships with depth
//	GET /departments/123?_expand=all&_depth=3
//
//	// Select specific fields only
//	GET /users/123?_select=id,name,email
//
//	// Bypass cache for fresh data
//	GET /users/123?_nocache=true
//
//	// Combine expansion with field selection
//	GET /users/123?_expand=department&_select=id,name,department&_depth=1
func GetFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_GET)
		defer span.End()

		cctx := types.NewControllerContext(c)
		log := logger.Controller.WithControllerContext(cctx, consts.PHASE_GET)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_GET)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var err error
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if err = c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
				log.Error(err)
				ResponseJSON(c, CodeInvalidParam.WithErr(err))
				otel.RecordError(span, err)
				return
			}
			logRequest(log, consts.PHASE_GET, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_GET, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_GET)
				return svc.Get(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_GET, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		var param string
		if len(cfg) > 0 {
			param = cctx.Params[util.Deref(cfg[0]).ParamName]
		}
		if len(param) == 0 {
			log.Error(CodeNotFoundRouteParam)
			ResponseJSON(c, CodeNotFoundRouteParam)
			otel.RecordError(span, errors.New(CodeNotFoundRouteParam.Msg()))
			return
		}
		index, _ := c.GetQuery(consts.QUERY_INDEX)
		selects, _ := c.GetQuery(consts.QUERY_SELECT)

		// The underlying type of interface types.Model must be pointer to structure, such as *model.User.
		// 'typ' is the structure type, such as: model.User.
		// 'm' is the structure value, such as: &model.User{ID: myid, Name: myname}.
		typ := reflect.TypeOf(*new(M)).Elem()
		m := reflect.New(typ).Interface().(M) //nolint:errcheck
		m.SetID(param)                        // `GetBefore` hook need id.

		var err error
		var expands []string
		nocache := true // default disable cache.
		depth := 1
		if nocacheStr, ok := c.GetQuery(consts.QUERY_NOCACHE); ok {
			var _nocache bool
			if _nocache, err = strconv.ParseBool(nocacheStr); err == nil {
				nocache = _nocache
			}
		}
		if depthStr, ok := c.GetQuery(consts.QUERY_DEPTH); ok {
			depth, _ = strconv.Atoi(depthStr)
			if depth < 1 || depth > 99 {
				depth = 1
			}
		}
		if expandStr, ok := c.GetQuery(consts.QUERY_EXPAND); ok {
			var _expands []string
			items := strings.Split(expandStr, ",")
			if len(items) > 0 {
				if items[0] == consts.VALUE_ALL { // expand all feilds
					items = m.Expands()
				}
			}
			for _, e := range m.Expands() {
				for _, item := range items {
					if strings.EqualFold(item, e) {
						_expands = append(_expands, e)
					}
				}
			}
			// fmt.Println("_expends: ", _expands)
			fieldsMap := make(map[string]reflect.Kind)
			for i := range typ.NumField() {
				fieldsMap[typ.Field(i).Name] = typ.Field(i).Type.Kind()
			}
			for _, e := range _expands {
				// If the expanding field not exists in the structure fiedls, skip depth expand.
				// TODO: if the field type is the structure name, make depth expand.
				kind, found := fieldsMap[e]
				if !found {
					expands = append(expands, e)
					continue
				}
				// If the expanding field exists in the structure but the kind is not slice, skip depth expand.
				if kind != reflect.Slice {
					expands = append(expands, e)
					continue
				}
				t := make([]string, depth)
				for i := range depth {
					t[i] = e
				}
				// If expand="Children" and depth=3, the depth expanded is "Children.Children.Children"
				expands = append(expands, strings.Join(t, "."))
			}
			// fmt.Println("expands: ", expands)
		}
		log.Infoz("", zap.Object(typ.Name(), m))

		// 1.Perform business logic processing before get resource.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_GET_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_GET_BEFORE)
			return svc.GetBefore(serviceCtxBefore, m)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		// 2.Get resource from database.
		if err = handler(types.NewDatabaseContext(c)).
			WithIndex(index).
			WithSelect(strings.Split(selects, ",")...).
			WithExpand(expands).
			WithCache(!nocache).
			Get(m, param); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// 3.Perform business logic processing after get resource.
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_GET_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_GET_AFTER)
			return svc.GetAfter(serviceCtxAfter, m)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}
		// It will returns a empty types.Model if found nothing from database,
		// we should response status code "CodeNotFound".
		if len(m.GetID()) == 0 || (m.GetCreatedAt().Equal(time.Time{})) {
			log.Error(CodeNotFound)
			ResponseJSON(c, CodeNotFound)
			otel.RecordError(span, errors.New(CodeNotFound.Msg()))
			return
		}

		// 4.record operation log to database.
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_GET,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		if err = am.RecordOperation(types.NewDatabaseContext(c), m, &modellogmgmt.OperationLog{
			OP:        consts.OP_GET,
			Model:     typ.Name(),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		ResponseJSON(c, CodeSuccess, m)
	}
}

// BatchCreate
// Example:
/*
Request Method: POST
Request URL: /api/v1/users/batch

Request Data:
{
  "items": [
    {
      "username": "johndoe",
      "email": "john.doe@example.com",
      "firstName": "John",
      "lastName": "Doe",
      "department": "Engineering"
    },
    {
      "username": "janedoe",
      "email": "jane.doe@example.com",
      "firstName": "Jane",
      "lastName": "Doe",
      "department": "Marketing"
    },
    {
      "username": "bobsmith",
      "email": "bob.smith@example.com",
      "firstName": "Bob",
      "lastName": "Smith",
      "department": "Finance"
    }
  ],
  "options": {
    "continueOnError": true
  }
}

Response Data:
{
  "items": [
    {
      "id": "user-123",
      "username": "johndoe",
      "email": "john.doe@example.com",
      "firstName": "John",
      "lastName": "Doe",
      "department": "Engineering",
      "createdAt": "2025-03-06T10:15:30Z"
    },
    {
      "status": "error",
      "error": {
        "code": 400,
        "message": "Email already in use"
      },
      "request": {
        "username": "janedoe",
        "email": "jane.doe@example.com",
        "firstName": "Jane",
        "lastName": "Doe",
        "department": "Marketing"
      }
    },
    {
      "id": "user-125",
      "username": "bobsmith",
      "email": "bob.smith@example.com",
      "firstName": "Bob",
      "lastName": "Smith",
      "department": "Finance",
      "createdAt": "2025-03-06T10:15:30Z"
    }
  ],
  "summary": {
    "total": 3,
    "succeeded": 2,
    "failed": 1
  }
}
}
*/

type requestData[M types.Model] struct {
	// Ids is the id list that should be batch delete.
	Ids []string `json:"ids,omitempty"`
	// Items is the resource list that should be batch create/update/partial update.
	Items []M `json:"items,omitempty"`
	// Options is the batch operation options.
	Options *options `json:"options,omitempty"`
	// Summary is the batch operation result summary.
	Summary *summary `json:"summary,omitempty"`
}

type options struct {
	Atomic bool `json:"atomic,omitempty"`
	Purge  bool `json:"purge,omitempty"`
}

type summary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

func CreateMany[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	CreateManyFactory[M, REQ, RSP]()(c)
}

// CreateManyFactory is a factory function that produces a gin handler for batch creating multiple resources.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles batch resource creation in database
//   - Service hooks (CreateManyBefore/CreateManyAfter) are executed for business logic
//   - Processing flow: Request -> ServiceBefore -> Database -> ServiceAfter -> Response
//   - The request body is bound to requestData[M] structure containing Items slice
//   - Automatic setting of CreatedBy/UpdatedBy fields from context for each item
//   - Supports atomic operations through options configuration
//   - Operation logging is automatically recorded to database
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over batch resource creation
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.CreateMany -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic and database operations
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP POST requests
//
// HTTP Response:
//   - Success: 201 Created with batch operation summary and created resources
//   - Error: 400 Bad Request for invalid parameters, 500 Internal Server Error for other failures
//
// Request Body Format (Unified Types):
//
//	{
//	  "items": [/* array of resources to create */],
//	  "options": {
//	    "atomic": true  // optional: whether to perform atomic operation
//	  }
//	}
//
// Response Format (Unified Types):
//
//	{
//	  "items": [/* array of created resources */],
//	  "summary": {
//	    "total": 10,
//	    "succeeded": 10,
//	    "failed": 0
//	  }
//	}
//
// Examples:
//
// Unified types (automatic mode):
//
//	CreateManyFactory[*model.User, *model.User, *model.User]()
//
// Custom types (manual mode):
//
//	CreateManyFactory[*model.User, *CreateManyUsersRequest, *CreateManyUsersResponse]()
//	// Service layer controls all batch creation logic
func CreateManyFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var err error
		var reqErr error

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_CREATE_MANY)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_CREATE_MANY)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_CREATE_MANY)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_CREATE_MANY, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_CREATE_MANY, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_CREATE_MANY)
				return svc.CreateMany(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_CREATE_MANY, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		var req requestData[M]
		typ := reflect.TypeOf(*new(M)).Elem()
		val := reflect.New(typ).Interface().(M) //nolint:errcheck
		if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
			log.Error(reqErr)
			ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
			otel.RecordError(span, err)
			return
		}
		if errors.Is(reqErr, io.EOF) {
			log.Warn(ErrRequestBodyEmpty)
		}
		logRequest(log, consts.PHASE_CREATE_MANY, req)

		if req.Options == nil {
			req.Options = new(options)
		}
		for _, m := range req.Items {
			m.SetCreatedBy(c.GetString(consts.CTX_USERNAME))
			m.SetUpdatedBy(c.GetString(consts.CTX_USERNAME))
			log.Infoz("create_many", zap.Bool("atomic", req.Options.Atomic), zap.Object(typ.Name(), m))
		}

		// 1.Perform business logic processing before batch create resource.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_CREATE_MANY_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_CREATE_MANY_BEFORE)
			return svc.CreateManyBefore(serviceCtxBefore, req.Items...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}

		// 2.Batch create resource in database.
		if !errors.Is(reqErr, io.EOF) {
			if err = handler(types.NewDatabaseContext(c)).WithExpand(val.Expands()).Create(req.Items...); err != nil {
				log.Error(err)
				ResponseJSON(c, CodeFailure.WithErr(err))
				otel.RecordError(span, err)
				return
			}
		}
		// 3.Perform business logic processing after batch create resource
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_CREATE_MANY_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_CREATE_MANY_AFTER)
			return svc.CreateManyAfter(serviceCtxAfter, req.Items...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		record, _ := json.Marshal(req)
		reqData, _ := json.Marshal(req)
		respData, _ := json.Marshal(req)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_CREATE_MANY,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	Record:    util.BytesToString(record),
		// 	Request:   util.BytesToString(reqData),
		// 	Response:  util.BytesToString(respData),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		if err = am.RecordOperation(types.NewDatabaseContext(c), val, &modellogmgmt.OperationLog{
			OP:        consts.OP_CREATE_MANY,
			Model:     typ.Name(),
			Record:    util.BytesToString(record),
			Request:   util.BytesToString(reqData),
			Response:  util.BytesToString(respData),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		// FIXME: 如果某些字段增加了 gorm unique tag, 则更新成功后的资源 ID 时随机生成的，并不是数据库中的
		if !errors.Is(reqErr, io.EOF) {
			req.Summary = &summary{
				Total:     len(req.Items),
				Succeeded: len(req.Items),
				Failed:    0,
			}
		}
		logResponse(log, consts.PHASE_CREATE_MANY, req)
		ResponseJSON(c, CodeSuccess.WithStatus(http.StatusCreated), req)
	}
}

// DeleteMany
func DeleteMany[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	DeleteManyFactory[M, REQ, RSP]()(c)
}

// DeleteManyFactory is a factory function that produces a gin handler for batch deleting multiple resources.
// It supports two different processing modes based on the type relationship between M, REQ, and RSP:
//
// Mode 1: Unified Types (M == REQ == RSP)
// When all three generic types are identical, the factory enables automatic resource management:
//   - Controller layer automatically handles batch resource deletion from database
//   - Service hooks (DeleteManyBefore/DeleteManyAfter) are executed for business logic
//   - Processing flow: Request -> ServiceBefore -> Database -> ServiceAfter -> Response
//   - The request body is bound to requestData[M] structure containing Ids slice
//   - Resources are identified by their IDs and converted to model instances for processing
//   - Supports soft delete and hard delete (purge) through options configuration
//   - Operation logging is automatically recorded to database
//
// Mode 2: Custom Types (M != REQ or REQ != RSP)
// When types differ, the factory delegates full control to the service layer:
//   - Service layer has complete control over batch resource deletion
//   - No automatic database operations or service hooks
//   - Processing flow: Request -> Service.DeleteMany -> Response
//   - The request body is bound to the REQ type
//   - Service must handle all business logic and database operations
//
// Type Parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct, e.g., *User)
//   - REQ: Request type that implements types.Request interface
//   - RSP: Response type that implements types.Response interface
//
// Parameters:
//   - cfg: Optional controller configuration for customizing database handler
//
// Returns:
//   - gin.HandlerFunc: A gin handler function for HTTP DELETE requests
//
// HTTP Response:
//   - Success: 204 No Content (unified types) or 200 OK (custom types) with optional response data
//   - Error: 400 Bad Request for invalid parameters, 404 Not Found if resource doesn't exist, 500 Internal Server Error for other failures
//
// Request Body Format (Unified Types):
//
//	{
//	  "ids": ["id1", "id2", "id3"],  // array of resource IDs to delete
//	  "options": {
//	    "purge": false  // optional: true for hard delete, false for soft delete (default)
//	  }
//	}
//
// Response Format (Unified Types):
//
//	HTTP 204 No Content (empty response body)
//
// Examples:
//
// Unified types (automatic mode):
//
//	DeleteManyFactory[*model.User, *model.User, *model.User]()
//	// DELETE /users with {"ids": ["user1", "user2"], "options": {"purge": false}}
//
// Custom types (manual mode):
//
//	DeleteManyFactory[*model.User, *DeleteManyUsersRequest, *DeleteManyUsersResponse]()
//	// Service layer controls all batch deletion logic
func DeleteManyFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var err error
		var reqErr error

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_DELETE_MANY)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_DELETE_MANY)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_DELETE_MANY)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeInvalidParam.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_DELETE_MANY, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_DELETE_MANY, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_DELETE_MANY)
				return svc.DeleteMany(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_DELETE_MANY, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		var req requestData[M]
		if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
			log.Error(reqErr)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		if errors.Is(reqErr, io.EOF) {
			log.Warn(ErrRequestBodyEmpty)
		}

		// 1.Perform business logic processing before batch delete resources.
		typ := reflect.TypeOf(*new(M)).Elem()
		req.Items = make([]M, 0, len(req.Ids))
		for _, id := range req.Ids {
			m := reflect.New(typ).Interface().(M) //nolint:errcheck
			m.SetID(id)
			req.Items = append(req.Items, m)
		}
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_DELETE_MANY_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_DELETE_MANY_BEFORE)
			return svc.DeleteManyBefore(serviceCtxBefore, req.Items...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		if req.Options == nil {
			req.Options = new(options)
		}
		// 2.Batch delete resources in database.
		if !errors.Is(reqErr, io.EOF) {
			// purge mode is current not allowed in request.
			//
			// if err = handler(types.NewDatabaseContext(c)).WithPurge(req.Options.Purge).Delete(req.Items...); err != nil {
			if err = handler(types.NewDatabaseContext(c)).Delete(req.Items...); err != nil {
				log.Error(err)
				ResponseJSON(c, CodeFailure.WithErr(err))
				otel.RecordError(span, err)
				return
			}
		}
		// 3.Perform business logic processing after batch delete resources.
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_DELETE_MANY_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_DELETE_MANY_AFTER)
			return svc.DeleteManyAfter(serviceCtxBefore, req.Items...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		record, _ := json.Marshal(req)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_DELETE_MANY,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	Record:    util.BytesToString(record),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		m := reflect.New(typ).Interface().(M) //nolint:errcheck
		if err = am.RecordOperation(types.NewDatabaseContext(c), m, &modellogmgmt.OperationLog{
			OP:        consts.OP_DELETE_MANY,
			Model:     typ.Name(),
			Record:    util.BytesToString(record),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		if !errors.Is(reqErr, io.EOF) {
			req.Summary = &summary{
				Total:     len(req.Items),
				Succeeded: len(req.Items),
				Failed:    0,
			}
		}
		req.Ids = nil
		req.Items = nil
		req.Options = nil
		// not response req.
		ResponseJSON(c, CodeSuccess.WithStatus(http.StatusNoContent))
	}
}

// UpdateMany
func UpdateMany[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	UpdateManyFactory[M, REQ, RSP]()(c)
}

// UpdateManyFactory is a generic factory function that creates a Gin handler for batch updating resources.
// It supports two processing modes based on type relationships:
//
// 1. **Unified Types Mode** (M == REQ == RSP):
//   - All type parameters represent the same underlying type
//   - Automatic database operations and service hooks are executed
//   - Request body is parsed as requestData[M] containing Items array
//   - Automatic field population (UpdatedBy) and operation logging
//   - Four-phase processing: pre-update hooks → database update → post-update hooks → operation logging
//
// 2. **Custom Types Mode** (M != REQ or REQ != RSP):
//   - Different types for Model, Request, and Response
//   - Service layer controls all update logic through UpdateMany method
//   - No automatic database operations - service handles everything
//   - Custom request/response type handling
//
// Type Parameters:
//   - M: Model type implementing types.Model interface (database entity)
//   - REQ: Request type implementing types.Request interface (input data structure)
//   - RSP: Response type implementing types.Response interface (output data structure)
//
// Request Format (Unified Types Mode):
//   - Content-Type: application/json
//   - Body: {"items": [M, M, ...], "options": {"atomic": bool}}
//   - Empty body is allowed (treated as no-op)
//
// Response Format:
//   - Success (200): Updated resource data with operation summary
//   - Error (400): Invalid request parameters
//   - Error (500): Internal server error
//
// Automatic Features (Unified Types Mode):
//   - Service hooks: UpdateManyBefore/UpdateManyAfter for business logic
//   - Database batch update operations
//   - Operation logging with request/response capture
//   - Error handling and logging throughout all phases
//   - Request validation and empty body handling
//
// The function handles both empty and non-empty request bodies gracefully,
// with comprehensive error handling and detailed operation logging for audit purposes.
func UpdateManyFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var err error
		var reqErr error

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_UPDATE_MANY)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_UPDATE_MANY)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_UPDATE_MANY)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeFailure.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_UPDATE_MANY, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_UPDATE_MANY, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_UPDATE_MANY)
				return svc.UpdateMany(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_UPDATE_MANY, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		var req requestData[M]
		if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
			log.Error(reqErr)
			ResponseJSON(c, CodeFailure.WithErr(reqErr))
			otel.RecordError(span, err)
			return
		}
		if errors.Is(reqErr, io.EOF) {
			log.Warn(ErrRequestBodyEmpty)
		}
		logRequest(log, consts.PHASE_UPDATE_MANY, req)

		// 1.Perform business logic processing before batch update resource.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_UPDATE_MANY_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_UPDATE_MANY_BEFORE)
			return svc.UpdateManyBefore(serviceCtxBefore, req.Items...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		// 2.Batch update resource in database.
		if !errors.Is(reqErr, io.EOF) {
			if err = handler(types.NewDatabaseContext(c)).Update(req.Items...); err != nil {
				log.Error(err)
				ResponseJSON(c, CodeFailure.WithErr(err))
				otel.RecordError(span, err)
				return
			}
		}
		// 3.Perform business logic processing after batch update resource.
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_UPDATE_MANY_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_UPDATE_MANY_AFTER)
			return svc.UpdateManyAfter(serviceCtxAfter, req.Items...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		typ := reflect.TypeOf(*new(M)).Elem()
		record, _ := json.Marshal(req)
		reqData, _ := json.Marshal(req)
		respData, _ := json.Marshal(req)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_UPDATE_MANY,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	Record:    util.BytesToString(record),
		// 	Request:   util.BytesToString(reqData),
		// 	Response:  util.BytesToString(respData),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		m := reflect.New(typ).Interface().(M) //nolint:errcheck
		if err = am.RecordOperation(types.NewDatabaseContext(c), m, &modellogmgmt.OperationLog{
			OP:        consts.OP_UPDATE_MANY,
			Model:     typ.Name(),
			Record:    util.BytesToString(record),
			Request:   util.BytesToString(reqData),
			Response:  util.BytesToString(respData),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		if !errors.Is(reqErr, io.EOF) {
			req.Summary = &summary{
				Total:     len(req.Items),
				Succeeded: len(req.Items),
				Failed:    0,
			}
		}
		logResponse(log, consts.PHASE_UPDATE_MANY, req)
		ResponseJSON(c, CodeSuccess, req)
	}
}

// PatchMany
func PatchMany[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	PatchManyFactory[M, REQ, RSP]()(c)
}

// PatchManyFactory is a generic factory function that creates a Gin handler for batch partial updating resources.
// It supports two processing modes based on type relationships:
//
// 1. **Unified Types Mode** (M == REQ == RSP):
//   - All type parameters represent the same underlying type
//   - Automatic database operations and service hooks are executed
//   - Request body is parsed as requestData[M] containing Items array
//   - Performs field-level partial updates by comparing existing and new values
//   - Four-phase processing: pre-patch hooks → database update → post-patch hooks → operation logging
//
// 2. **Custom Types Mode** (M != REQ or REQ != RSP):
//   - Different types for Model, Request, and Response
//   - Service layer controls all patch logic through PatchMany method
//   - No automatic database operations - service handles everything
//   - Custom request/response type handling
//
// Type Parameters:
//   - M: Model type implementing types.Model interface (database entity)
//   - REQ: Request type implementing types.Request interface (input data structure)
//   - RSP: Response type implementing types.Response interface (output data structure)
//
// Request Format (Unified Types Mode):
//   - Content-Type: application/json
//   - Body: {"items": [M, M, ...], "options": {"atomic": bool}}
//   - Empty body is allowed (treated as no-op)
//   - Only non-zero fields in request items will be updated (partial update behavior)
//
// Response Format:
//   - Success (200): Updated resource data with operation summary
//   - Error (400): Invalid request parameters
//   - Error (500): Internal server error
//
// Automatic Features (Unified Types Mode):
//   - Service hooks: PatchManyBefore/PatchManyAfter for business logic
//   - Field-level partial update logic with existing record retrieval
//   - Database batch update operations
//   - Operation logging with request/response capture
//   - Error handling and logging throughout all phases
//   - Request validation and empty body handling
//
// The function handles both empty and non-empty request bodies gracefully,
// with comprehensive error handling and detailed operation logging for audit purposes.
// Unlike UpdateMany, this function performs partial updates by only modifying non-zero fields.
func PatchManyFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		var err error
		var reqErr error

		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_PATCH_MANY)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_PATCH_MANY)
		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_PATCH_MANY)

		if !model.AreTypesEqual[M, REQ, RSP]() {
			var req REQ
			var rsp RSP

			reqTyp := reflect.TypeFor[REQ]()
			switch reqTyp.Kind() {
			case reflect.Struct:
				req = reflect.New(reqTyp).Elem().Interface().(REQ) //nolint:errcheck
			case reflect.Pointer:
				for reqTyp.Kind() == reflect.Pointer {
					reqTyp = reqTyp.Elem()
				}
				req = reflect.New(reqTyp).Interface().(REQ) //nolint:errcheck
			}

			if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
				log.Error(reqErr)
				ResponseJSON(c, CodeFailure.WithErr(reqErr))
				otel.RecordError(span, err)
				return
			}
			if errors.Is(reqErr, io.EOF) {
				log.Warn(ErrRequestBodyEmpty)
			}
			logRequest(log, consts.PHASE_PATCH_MANY, req)
			var serviceCtx *types.ServiceContext
			if rsp, err = traceServiceOperation[M, RSP](ctrlSpanCtx, consts.PHASE_PATCH_MANY, func(spanCtx context.Context) (RSP, error) {
				serviceCtx = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_PATCH_MANY)
				return svc.PatchMany(serviceCtx, req)
			}); err != nil {
				log.Error(err)
				handleServiceError(c, serviceCtx, err)
				otel.RecordError(span, err)
				return
			}
			logResponse(log, consts.PHASE_PATCH_MANY, rsp)
			ResponseJSON(c, CodeSuccess, rsp)
			return
		}

		var req requestData[M]
		var shouldUpdates []M
		typ := reflect.TypeOf(*new(M)).Elem()
		if reqErr = c.ShouldBindJSON(&req); reqErr != nil && !errors.Is(reqErr, io.EOF) {
			log.Error(reqErr)
			ResponseJSON(c, CodeFailure.WithErr(reqErr))
			otel.RecordError(span, err)
			return
		}
		if errors.Is(reqErr, io.EOF) {
			log.Warn(ErrRequestBodyEmpty)
		}
		logRequest(log, consts.PHASE_PATCH_MANY, req)
		for _, m := range req.Items {
			var results []M
			v := reflect.New(typ).Interface().(M) //nolint:errcheck
			v.SetID(m.GetID())
			if err = handler(types.NewDatabaseContext(c)).WithLimit(1).WithQuery(v).List(&results); err != nil {
				log.Error(err)
				otel.RecordError(span, err)
				continue
			}
			if len(results) != 1 {
				log.Warnf(fmt.Sprintf("partial update resource not found, expect 1 but got: %d", len(results)))
				continue
			}
			if len(results[0].GetID()) == 0 {
				log.Warnf("partial update resource not found, id is empty")
				continue
			}
			oldVal, newVal := reflect.ValueOf(results[0]).Elem(), reflect.ValueOf(m).Elem()
			patchValue(log, typ, oldVal, newVal)
			shouldUpdates = append(shouldUpdates, oldVal.Addr().Interface().(M)) //nolint:errcheck
		}

		// 1.Perform business logic processing before batch patch resource.
		var serviceCtxBefore *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_PATCH_MANY_BEFORE, func(spanCtx context.Context) error {
			serviceCtxBefore = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_PATCH_MANY_BEFORE)
			return svc.PatchManyBefore(serviceCtxBefore, shouldUpdates...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxBefore, err)
			otel.RecordError(span, err)
			return
		}
		// 2.Batch partial update resource in database.
		if !errors.Is(reqErr, io.EOF) {
			if err = handler(types.NewDatabaseContext(c)).Update(shouldUpdates...); err != nil {
				log.Error(err)
				ResponseJSON(c, CodeFailure.WithErr(err))
				otel.RecordError(span, err)
				return
			}
		}
		// 3.Perform business logic processing after batch patch resource.
		var serviceCtxAfter *types.ServiceContext
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_PATCH_MANY_AFTER, func(spanCtx context.Context) error {
			serviceCtxAfter = types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_PATCH_MANY_AFTER)
			return svc.PatchManyAfter(serviceCtxAfter, shouldUpdates...)
		}); err != nil {
			log.Error(err)
			handleServiceError(c, serviceCtxAfter, err)
			otel.RecordError(span, err)
			return
		}

		// 4.record operation log to database.
		// NOTE: We should record the `req` instead of `oldVal`, the req is `newVal`.
		record, _ := json.Marshal(req)
		reqData, _ := json.Marshal(req)
		respData, _ := json.Marshal(req)
		// cb.Enqueue(&modellogmgmt.OperationLog{
		// 	OP:        consts.OP_PATCH_MANY,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	Record:    util.BytesToString(record),
		// 	Request:   util.BytesToString(reqData),
		// 	Response:  util.BytesToString(respData),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestID: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// })
		m := reflect.New(typ).Interface().(M) //nolint:errcheck
		if err = am.RecordOperation(types.NewDatabaseContext(c), m, &modellogmgmt.OperationLog{
			OP:        consts.OP_PATCH_MANY,
			Model:     typ.Name(),
			Record:    util.BytesToString(record),
			Request:   util.BytesToString(reqData),
			Response:  util.BytesToString(respData),
			IP:        c.ClientIP(),
			User:      c.GetString(consts.CTX_USERNAME),
			RequestID: c.GetString(consts.REQUEST_ID),
			URI:       c.Request.RequestURI,
			Method:    c.Request.Method,
			UserAgent: c.Request.UserAgent(),
		}); err != nil {
			log.Warn(err)
		}

		if !errors.Is(reqErr, io.EOF) {
			req.Summary = &summary{
				Total:     len(req.Items),
				Succeeded: len(req.Items),
				Failed:    0,
			}
		}
		logResponse(log, consts.PHASE_PATCH_MANY, req)
		ResponseJSON(c, CodeSuccess, req)
	}
}

// Export is a generic function to product gin handler to export resources to frontend.
// The resource type deponds on the type of interface types.Model.
//
// If you want make a structure field as query parameter, you should add a "schema"
// tag for it. for example: schema:"name"
//
// TODO:combine query parameter 'page' and 'size' into decoded types.Model
// FIX: retrieve records recursive (current not support in gorm.)
// https://stackoverflow.com/questions/69395891/get-recursive-field-values-in-gorm
// DB.Preload("Category.Category.Category").Find(&Category)
// its works for me.
//
// Query parameters:
//   - All feilds of types.Model's underlying structure but excluding some special fields,
//     such as "password", field value too large, json tag is "-", etc.
//   - `_expand`: strings (multiple items separated by ",").
//     The responsed data to frontend will expanded(retrieve data from external table accoding to foreign key)
//     For examples:
//     /department/myid?_expand=children
//     /department/myid?_expand=children,parent
//   - `_depth`: strings or interger.
//     How depth to retrieve records from datab recursively, default to 1, value scope is [1,99].
//     For examples:
//     /department/myid?_expand=children&_depth=3
//     /department/myid?_expand=children,parent&_depth=10
//   - `_fuzzy`: bool
//     fuzzy match records in database, default to fase.
//     For examples:
//     /department/myid?_fuzzy=true
func Export[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	ExportFactory[M, REQ, RSP]()(c)
}

// ExportFactory is a factory function to export resources to frontend.
func ExportFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_EXPORT)
		defer span.End()

		var page, size, limit int
		var startTime, endTime time.Time
		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_EXPORT)
		if pageStr, ok := c.GetQuery(consts.QUERY_PAGE); ok {
			page, _ = strconv.Atoi(pageStr)
		}
		if sizeStr, ok := c.GetQuery(consts.QUERY_SIZE); ok {
			size, _ = strconv.Atoi(sizeStr)
		}
		if limitStr, ok := c.GetQuery(consts.QUERY_LIMIT); ok {
			limit, _ = strconv.Atoi(limitStr)
		}
		columnName, _ := c.GetQuery(consts.QUERY_COLUMN_NAME)
		index, _ := c.GetQuery(consts.QUERY_INDEX)
		selects, _ := c.GetQuery(consts.QUERY_SELECT)
		if startTimeStr, ok := c.GetQuery(consts.QUERY_START_TIME); ok {
			startTime, _ = time.ParseInLocation(consts.DATE_TIME_LAYOUT, startTimeStr, time.Local)
		}
		if endTimeStr, ok := c.GetQuery(consts.QUERY_END_TIME); ok {
			endTime, _ = time.ParseInLocation(consts.DATE_TIME_LAYOUT, endTimeStr, time.Local)
		}

		// The underlying type of interface types.Model must be pointer to structure, such as *model.User.
		// 'typ' is the structure type, such as: model.User.
		// 'm' is the structure value, such as: &model.User{ID: myid, Name: myname}.
		typ := reflect.TypeOf(*new(M)).Elem() // the real underlying structure type
		m := reflect.New(typ).Interface().(M) //nolint:errcheck

		if err := schema.NewDecoder().Decode(m, c.Request.URL.Query()); err != nil {
			log.Warn("failed to parse uri query parameter into model: ", err)
		}
		log.Info("query parameter: ", m)

		var err error
		var or bool
		var fuzzy bool
		depth := 1
		var expands []string
		data := make([]M, 0)
		if orStr, ok := c.GetQuery(consts.QUERY_OR); ok {
			or, _ = strconv.ParseBool(orStr)
		}
		if fuzzyStr, ok := c.GetQuery(consts.QUERY_FUZZY); ok {
			fuzzy, _ = strconv.ParseBool(fuzzyStr)
		}
		if depthStr, ok := c.GetQuery(consts.QUERY_DEPTH); ok {
			depth, _ = strconv.Atoi(depthStr)
			if depth < 1 || depth > 99 {
				depth = 1
			}
		}
		if expandStr, ok := c.GetQuery(consts.QUERY_EXPAND); ok {
			var _expands []string
			items := strings.Split(expandStr, ",")
			if len(items) > 0 {
				if items[0] == consts.VALUE_ALL { // expand all feilds
					items = m.Expands()
				}
			}
			for _, e := range m.Expands() {
				for _, item := range items {
					if strings.EqualFold(item, e) {
						_expands = append(_expands, e)
					}
				}
			}
			// fmt.Println("_expends: ", _expands)
			fieldsMap := make(map[string]reflect.Kind)
			for i := range typ.NumField() {
				fieldsMap[typ.Field(i).Name] = typ.Field(i).Type.Kind()
			}
			for _, e := range _expands {
				// If the expanding field not exists in the structure fiedls, skip depth expand.
				kind, found := fieldsMap[e]
				if !found {
					expands = append(expands, e)
					continue
				}
				// If the expanding field exists in the structure but the kind is not slice, skip depth expand.
				if kind != reflect.Slice {
					expands = append(expands, e)
					continue
				}
				t := make([]string, depth)
				for i := range depth {
					t[i] = e
				}
				// fmt.Println("t: ", t)
				// If expand="Children" and depth=3, the depth expanded is "Children.Children.Children"
				expands = append(expands, strings.Join(t, "."))
			}
			// fmt.Println("expands: ", expands)
		}

		svc := service.Factory[M, REQ, RSP]().Service(consts.PHASE_EXPORT)
		svcCtx := types.NewServiceContext(c)
		// 1.Perform business logic processing before list resources.
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_EXPORT, func(spanCtx context.Context) error {
			return svc.ListBefore(types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_EXPORT), &data)
		}); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		sortBy, _ := c.GetQuery(consts.QUERY_SORTBY)
		_, _ = page, size
		// 2.List resources from database.
		if err = handler(types.NewDatabaseContext(c)).
			// WithPagination(page, size). // 不要使用 WithPagination, 否则 WithLimit 不生效
			WithLimit(limit).
			WithIndex(index).
			WithSelect(strings.Split(selects, ",")...).
			WithQuery(svc.Filter(svcCtx, m), types.QueryConfig{
				FuzzyMatch: fuzzy,
				AllowEmpty: true,
				UseOr:      or,
				RawQuery:   svc.FilterRaw(svcCtx),
			}).
			WithExclude(m.Excludes()).
			WithExpand(expands, sortBy).
			WithOrder(sortBy).
			WithTimeRange(columnName, startTime, endTime).
			List(&data); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// 3.Perform business logic processing after list resources.
		if err = traceServiceHook[M](ctrlSpanCtx, consts.PHASE_EXPORT, func(spanCtx context.Context) error {
			return svc.ListAfter(types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_EXPORT), &data)
		}); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		log.Info("export data length: ", len(data))
		// 4.Export
		exported, err := traceServiceExport[M](ctrlSpanCtx, consts.PHASE_EXPORT, func(spanCtx context.Context) ([]byte, error) {
			return svc.Export(types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_EXPORT), data...)
		})
		if err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// // 5.record operation log to database.
		// var tableName string
		// items := strings.Split(typ.Name(), ".")
		// if len(items) > 0 {
		// 	tableName = pluralizeCli.Plural(strings.ToLower(items[len(items)-1]))
		// }
		// record, _ := json.Marshal(data)
		// if err := database.Database[*model.OperationLog]().WithDB(db).Create(&model.OperationLog{
		// 	Op:        model.OperationTypeExport,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	Record:    util.BytesToString(record),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestId: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// }); err != nil {
		// 	log.Error("failed to write operation log to database: ", err.Error())
		// }
		ResponseDATA(c, exported, map[string]string{
			"Content-Disposition": "attachment; filename=exported.xlsx",
		})
	}
}

// Import
func Import[M types.Model, REQ types.Request, RSP types.Response](c *gin.Context) {
	ImportFactory[M, REQ, RSP]()(c)
}

// ImportFactory
func ImportFactory[M types.Model, REQ types.Request, RSP types.Response](cfg ...*types.ControllerConfig[M]) gin.HandlerFunc {
	handler, _ := extractConfig(cfg...)
	return func(c *gin.Context) {
		ctrlSpanCtx, span := startControllerSpan[M](c, consts.PHASE_IMPORT)
		defer span.End()

		log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.PHASE_IMPORT)
		// NOTE:字段为 file 必须和前端协商好.
		file, err := c.FormFile("file")
		if err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// check file size.
		if file.Size > int64(MAX_IMPORT_SIZE) {
			log.Error(CodeTooLargeFile)
			ResponseJSON(c, CodeTooLargeFile)
			otel.RecordError(span, errors.New(CodeTooLargeFile.Msg()))
			return
		}
		fd, err := file.Open()
		if err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		defer fd.Close()

		buf := new(bytes.Buffer)
		if _, err = io.Copy(buf, fd); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// filetype must be png or jpg.
		filetype, mime := filetype.DetectBytes(buf.Bytes())
		_, _ = filetype, mime

		// check filetype

		ml, err := traceServiceImport(ctrlSpanCtx, consts.PHASE_IMPORT, func(spanCtx context.Context) ([]M, error) {
			return service.Factory[M, REQ, RSP]().Service(consts.PHASE_IMPORT).
				Import(types.NewServiceContext(c, spanCtx).WithPhase(consts.PHASE_IMPORT), buf)
		})
		if err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}

		// service layer already create/update the records in database, just update fields "created_by", "updated_by".
		for i := range ml {
			ml[i].SetCreatedBy(c.GetString(consts.CTX_USERNAME))
			ml[i].SetUpdatedBy(c.GetString(consts.CTX_USERNAME))
		}
		if err := handler(types.NewDatabaseContext(c)).Update(ml...); err != nil {
			log.Error(err)
			ResponseJSON(c, CodeFailure.WithErr(err))
			otel.RecordError(span, err)
			return
		}
		// // record operation log to database.
		// typ := reflect.TypeOf(*new(M)).Elem()
		// var tableName string
		// items := strings.Split(typ.Name(), ".")
		// if len(items) > 0 {
		// 	tableName = pluralizeCli.Plural(strings.ToLower(items[len(items)-1]))
		// }
		// record, _ := json.Marshal(ml)
		// if err := database.Database[*model.OperationLog]().WithDB(db).Create(&model.OperationLog{
		// 	Op:        model.OperationTypeImport,
		// 	Model:     typ.Name(),
		// 	Table:     tableName,
		// 	Record:    util.BytesToString(record),
		// 	IP:        c.ClientIP(),
		// 	User:      c.GetString(consts.CTX_USERNAME),
		// 	RequestId: c.GetString(consts.REQUEST_ID),
		// 	URI:       c.Request.RequestURI,
		// 	Method:    c.Request.Method,
		// 	UserAgent: c.Request.UserAgent(),
		// }); err != nil {
		// 	log.Error("failed to write operation log to database: ", err.Error())
		// }
		ResponseJSON(c, CodeSuccess)
	}
}
