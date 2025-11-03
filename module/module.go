// Package module provides a comprehensive module system for creating modular API endpoints
// with automatic CRUD operations, routing, and service layer integration.
//
// The module package enables developers to create reusable, self-contained modules
// that automatically register models, services, and routes with the framework.
// Each module encapsulates a complete API resource with customizable behavior.
//
// Key features:
//   - Automatic model registration with the ORM layer
//   - Service registration for business logic handling
//   - Dynamic route generation based on CRUD phases
//   - Support for both public and authenticated endpoints
//   - Flexible URL parameter customization
//   - Batch operation support for bulk operations
//
// Usage example:
//
//	// Define your module implementation
//	type HelloworldModule struct{}
//
//	func (HelloworldModule) Service() types.Service[*Helloworld, *Req, *Rsp] {
//	    return &HelloworldService{}
//	}
//	func (HelloworldModule) Pub() bool     { return false }
//	func (HelloworldModule) Route() string { return "hello-world" }
//	func (HelloworldModule) Param() string { return "id" }
//
//	// Register the module with desired CRUD operations
//	module.Use[*Helloworld, *Req, *Rsp, *HelloworldService](
//	    &HelloworldModule{},
//	    consts.PHASE_CREATE,
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	    consts.PHASE_UPDATE,
//	    consts.PHASE_DELETE,
//	)
//
// This automatically creates the following REST API endpoints:
//   - POST   /hello-world        (create single resource)
//   - GET    /hello-world        (list resources with pagination)
//   - GET    /hello-world/:id    (get single resource by ID)
//   - PUT    /hello-world/:id    (update single resource)
//   - DELETE /hello-world/:id    (delete single resource)
//
// Additional batch operations are available:
//   - POST   /hello-world/batch  (create multiple resources)
//   - PUT    /hello-world/batch  (update multiple resources)
//   - PATCH  /hello-world/batch  (patch multiple resources)
//   - DELETE /hello-world/batch  (delete multiple resources)
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

// Use registers a module with the framework, automatically setting up models, services, and routes
// for the specified CRUD phases. This is the main entry point for module registration.
//
// The function performs three main registration steps:
//  1. Model registration: Registers the model type with the ORM layer
//  2. Service registration: Registers the service for each specified phase
//  3. Route registration: Creates HTTP endpoints based on the module configuration and phases
//
// Generic type parameters:
//   - M: Model type that implements types.Model interface (must be pointer to struct)
//   - REQ: Request type for API operations (can be any serializable type)
//   - RSP: Response type for API operations (can be any serializable type)
//   - S: Service type that implements types.Service[M, REQ, RSP] interface
//
// Parameters:
//   - mod: The module instance that defines the API configuration
//   - phases: Variable number of CRUD phases to enable for this module
//
// Supported phases and their generated routes:
//   - PHASE_CREATE: POST /route (create single resource)
//   - PHASE_DELETE: DELETE /route/:param (delete single resource)
//   - PHASE_UPDATE: PUT /route/:param (update single resource)
//   - PHASE_PATCH: PATCH /route/:param (patch single resource)
//   - PHASE_LIST: GET /route (list resources with pagination)
//   - PHASE_GET: GET /route/:param (get single resource)
//   - PHASE_CREATE_MANY: POST /route/batch (create multiple resources)
//   - PHASE_DELETE_MANY: DELETE /route/batch (delete multiple resources)
//   - PHASE_UPDATE_MANY: PUT /route/batch (update multiple resources)
//   - PHASE_PATCH_MANY: PATCH /route/batch (patch multiple resources)
//
// Route processing:
//   - Automatically trims leading slashes and "api" prefix from module.Route()
//   - Uses module.Param() for URL parameter name, defaults to "id" if empty
//   - Registers routes as public or authenticated based on module.Pub()
//
// Example usage:
//
//	module.Use[*User, *UserRequest, *UserResponse, *UserService](
//	    &UserModule{},
//	    consts.PHASE_CREATE,
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	    consts.PHASE_UPDATE,
//	    consts.PHASE_DELETE,
//	)
func Use[M types.Model, REQ types.Request, RSP types.Response, S types.Service[M, REQ, RSP]](mod types.Module[M, REQ, RSP], phases ...consts.Phase) {
	// Register model with the ORM layer for database operations
	model.Register[M]()

	// Register service for each specified phase to handle business logic
	for _, p := range phases {
		service.Register[S](p)
	}

	// Process and normalize the route path
	// Remove leading slashes and "api" prefix to ensure consistent routing
	route := mod.Route()
	route = strings.TrimLeft(route, "/")
	route = strings.TrimLeft(route, "api")
	route = strings.TrimLeft(route, "/")

	// Get URL parameter name, default to "id" if not specified
	param := mod.Param()
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
