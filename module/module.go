// Package module provides a unified module registration system that automatically
// registers models, services, and HTTP routes for CRUD operations.
//
// The module system enables developers to create self-contained, reusable modules
// that encapsulate complete API resources with minimal boilerplate code. Each module
// defines its model, service, routing configuration, and authentication requirements.
//
// Core Components:
//   - Model: Database entity that implements types.Model interface
//   - Service: Business logic layer that implements types.Service interface
//   - Module: Configuration provider that implements types.Module interface
//
// Usage Pattern:
//
//  1. Define your model struct (embedding model.Base)
//  2. Define your request and response types
//  3. Implement a service that embeds service.Base
//  4. Implement a module that implements types.Module interface
//  5. Call module.Use() to register the module with desired CRUD phases
//
// Example - Full CRUD Module:
//
//	type User struct {
//	    Username string `json:"username"`
//	    Email    string `json:"email"`
//	    model.Base
//	}
//
//	type UserService struct {
//	    service.Base[*User, *User, *User]
//	}
//
//	type UserModule struct{}
//
//	func (UserModule) Service() types.Service[*User, *User, *User] {
//	    return &UserService{}
//	}
//	func (UserModule) Pub() bool     { return false }
//	func (UserModule) Route() string { return "users" }
//	func (UserModule) Param() string { return "id" }
//
//	func Register() {
//	    module.Use[*User, *User, *User, *UserService](
//	        &UserModule{},
//	        consts.PHASE_CREATE,
//	        consts.PHASE_DELETE,
//	        consts.PHASE_UPDATE,
//	        consts.PHASE_PATCH,
//	        consts.PHASE_LIST,
//	        consts.PHASE_GET,
//	        consts.PHASE_CREATE_MANY,
//	        consts.PHASE_DELETE_MANY,
//	        consts.PHASE_UPDATE_MANY,
//	        consts.PHASE_PATCH_MANY,
//	    )
//	}
//
// This will automatically create the following routes:
//   - POST   /users           (create)
//   - DELETE /users/:id       (delete)
//   - PUT    /users/:id       (update)
//   - PATCH  /users/:id       (patch)
//   - GET    /users           (list)
//   - GET    /users/:id       (get)
//   - POST   /users/batch     (create many)
//   - DELETE /users/batch     (delete many)
//   - PUT    /users/batch     (update many)
//   - PATCH  /users/batch     (patch many)
//
// Example - Read-Only Module:
//
//	module.Use[*LoginLog, *LoginLog, *LoginLog, *LoginLogService](
//	    &LoginLogModule{},
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	)
//
// This will create only:
//   - GET /loginlog           (list)
//   - GET /loginlog/:id       (get)
//
// Authentication:
//   - If Module.Pub() returns true: endpoints are publicly accessible
//   - If Module.Pub() returns false: endpoints require authentication/authorization
//
// Route Path Normalization:
//   - Leading slashes are automatically removed
//   - "api" prefix is automatically removed
//   - Route paths are normalized for consistency
//
// See module/helloworld and module/logger for complete working examples.
package module

import (
	"fmt"
	"strings"

	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/router"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
)

var notify = make(chan struct{})

// Init notify the module system that the framework is initialized
// Now the module system can start to register model, service, router, middleware.
func Init() error {
	close(notify)

	return nil
}

// Use registers a module with the framework, automatically setting up model registration,
// service registration, and HTTP route registration for the specified CRUD phases.
//
// This function is the primary entry point for module registration. It performs three
// main operations:
//  1. Registers the model type with the ORM layer for database operations
//  2. Registers the service type for each specified phase to handle business logic
//  3. Registers HTTP routes for each specified CRUD operation
//
// Generic Type Parameters:
//   - M: Model type that implements types.Model interface (typically a pointer to struct)
//   - REQ: Request type for API operations (can be any serializable type)
//   - RSP: Response type for API operations (can be any serializable type)
//   - S: Service type that implements types.Service[M, REQ, RSP] interface
//
// Parameters:
//   - mod: Module instance that implements types.Module[M, REQ, RSP] interface.
//     This provides configuration for routing, authentication, and service access.
//   - phases: Variable number of CRUD phases to register. Each phase corresponds to
//     a specific HTTP endpoint. Available phases:
//   - PHASE_CREATE:        POST   /route              (create single resource)
//   - PHASE_DELETE:        DELETE /route/:param        (delete single resource)
//   - PHASE_UPDATE:        PUT    /route/:param       (update single resource)
//   - PHASE_PATCH:         PATCH  /route/:param       (patch single resource)
//   - PHASE_LIST:          GET    /route              (list resources with pagination)
//   - PHASE_GET:           GET    /route/:param       (get single resource by ID)
//   - PHASE_CREATE_MANY:   POST   /route/batch         (create multiple resources)
//   - PHASE_DELETE_MANY:   DELETE /route/batch         (delete multiple resources)
//   - PHASE_UPDATE_MANY:   PUT    /route/batch         (update multiple resources)
//   - PHASE_PATCH_MANY:    PATCH  /route/batch         (patch multiple resources)
//
// Route Registration:
//   - Routes are automatically registered based on the module's Route() and Param() methods
//   - Route paths are normalized (leading slashes and "api" prefix are removed)
//   - URL parameter name defaults to "id" if Param() returns empty string
//   - Authentication is determined by the module's Pub() method
//
// Service Registration:
//   - Service is registered for each specified phase
//   - The same service instance is used for all phases, but lifecycle hooks are
//     called at appropriate times based on the phase
//
// Example Usage:
//
//	// Register a full CRUD module
//	module.Use[*User, *UserRequest, *UserResponse, *UserService](
//	    &UserModule{},
//	    consts.PHASE_CREATE,
//	    consts.PHASE_DELETE,
//	    consts.PHASE_UPDATE,
//	    consts.PHASE_PATCH,
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	)
//
//	// Register a read-only module
//	module.Use[*LoginLog, *LoginLog, *LoginLog, *LoginLogService](
//	    &LoginLogModule{},
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	)
//
// Note: This function must be called during application initialization, typically
// in a Register() function within your module package.
func Use[M types.Model, REQ types.Request, RSP types.Response, S types.Service[M, REQ, RSP]](mod types.Module[M, REQ, RSP], phases ...consts.Phase) {
	go func() {
		<-notify

		// Register model with the ORM layer for database operations
		model.Register[M]()

		// Register service for each specified phase to handle business logic
		for _, p := range phases {
			service.Register[S](p)
		}

		// Process and normalize the route path
		// Ensure consistent routing by trimming leading "/" and optional "api/" prefix.
		// Note: Use TrimPrefix instead of TrimLeft("api") to avoid removing
		//       any leading characters that happen to be in the set {'a','p','i'}.
		//       For example, "permissions" would incorrectly become "ermissions"
		//       with TrimLeft("api").
		route := mod.Route()
		route = strings.TrimPrefix(route, "/")    // trim leading slash
		route = strings.TrimPrefix(route, "api/") // trim optional "api/" prefix
		route = strings.TrimPrefix(route, "/")    // trim leading slash again if present

		// Get URL parameter name, default to "id" if not specified
		param := mod.Param()
		param = strings.TrimFunc(param, func(r rune) bool {
			return r == ' ' || r == '{' || r == '}' || r == '[' || r == ']' || r == ':'
		})
		if len(param) == 0 {
			param = "id"
		}

		// Register HTTP routes for each specified CRUD phase
		for _, p := range phases {
			switch p {
			case consts.PHASE_CREATE:
				// POST /route - Create single resource
				registerRouter(mod, route, nil, consts.Create)
			case consts.PHASE_DELETE:
				// DELETE /route/:param - Delete single resource by ID
				registerRouter(mod, fmt.Sprintf("%s/:%s", route, param), &types.ControllerConfig[M]{ParamName: param}, consts.Delete)
			case consts.PHASE_UPDATE:
				// PUT /route/:param - Update single resource by ID
				registerRouter(mod, fmt.Sprintf("%s/:%s", route, param), &types.ControllerConfig[M]{ParamName: param}, consts.Update)
			case consts.PHASE_PATCH:
				// PATCH /route/:param - Patch single resource by ID
				registerRouter(mod, fmt.Sprintf("%s/:%s", route, param), &types.ControllerConfig[M]{ParamName: param}, consts.Patch)
			case consts.PHASE_LIST:
				// GET /route - List resources with pagination
				registerRouter(mod, route, nil, consts.List)
			case consts.PHASE_GET:
				// GET /route/:param - Get single resource by ID
				registerRouter(mod, fmt.Sprintf("%s/:%s", route, param), &types.ControllerConfig[M]{ParamName: param}, consts.Get)
			case consts.PHASE_CREATE_MANY:
				// POST /route/batch - Create multiple resources
				registerRouter(mod, route+"/batch", nil, consts.CreateMany)
			case consts.PHASE_DELETE_MANY:
				// DELETE /route/batch - Delete multiple resources
				registerRouter(mod, route+"/batch", nil, consts.DeleteMany)
			case consts.PHASE_UPDATE_MANY:
				// PUT /route/batch - Update multiple resources
				registerRouter(mod, route+"/batch", nil, consts.UpdateMany)
			case consts.PHASE_PATCH_MANY:
				// PATCH /route/batch - Patch multiple resources
				registerRouter(mod, route+"/batch", nil, consts.PatchMany)
			}
		}
	}()
}

// registerRouter is a helper function that registers an HTTP route with the appropriate router
// based on the module's authentication requirements.
//
// This function determines whether to register the route with the public router (no authentication)
// or the authenticated router (requires authentication/authorization) based on the module's Pub() method.
//
// Generic type parameters:
//   - M: Model type that implements types.Model interface
//   - REQ: Request type for API operations
//   - RSP: Response type for API operations
//
// Parameters:
//   - mod: The module instance that defines authentication requirements
//   - route: The HTTP route path (e.g., "users", "users/:id", "users/batch")
//   - cfg: Controller configuration containing parameter names and other settings
//   - verb: HTTP verb/method for the route (GET, POST, PUT, DELETE, etc.)
//
// Route registration logic:
//   - If mod.Pub() returns true: registers with public router (no auth required)
//   - If mod.Pub() returns false: registers with authenticated router (auth required)
//
// This abstraction allows modules to easily control the authentication behavior
// of their endpoints without directly interacting with the router registration logic.
func registerRouter[M types.Model, REQ types.Request, RSP types.Response](mod types.Module[M, REQ, RSP], route string, cfg *types.ControllerConfig[M], verb consts.HTTPVerb) {
	if mod.Pub() {
		// Register with public router - no authentication required
		router.Register[M, REQ, RSP](router.Pub(), route, cfg, verb)
	} else {
		// Register with authenticated router - authentication/authorization required
		router.Register[M, REQ, RSP](router.Auth(), route, cfg, verb)
	}
}
