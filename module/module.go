// Package module provides a unified module registration system that automatically
// registers models, services, and HTTP routes for CRUD operations.
//
// A module consists of three components:
//   - Model: Database entity implementing types.Model
//   - Service: Business logic implementing types.Service
//   - Module: Configuration implementing types.Module
//
// Usage:
//  1. Define model (embedding model.Base), request/response types, and service (embedding service.Base)
//  2. Implement module with types.Module interface
//  3. Call module.Use() with desired CRUD phases
//
// Example:
//
//	module.Use[*User, *UserReq, *UserRsp, *UserService](
//	    &UserModule{},
//	    consts.PHASE_CREATE,
//	    consts.PHASE_LIST,
//	    consts.PHASE_GET,
//	)
//
// Route paths are normalized (leading slashes and "api/" prefix are removed).
// Authentication is controlled by Module.Pub() method.
//
// See module/helloworld for complete examples.
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

// Init notifies the module system that the framework is initialized.
// After calling Init, modules can start registering models, services, and routes.
func Init() error {
	close(notify)

	return nil
}

// Use registers a module with the framework, automatically setting up model,
// service, and HTTP route registration for the specified CRUD phases.
//
// Type Parameters:
//   - M: Model type implementing types.Model
//   - REQ: Request type for API operations
//   - RSP: Response type for API operations
//   - S: Service type implementing types.Service[M, REQ, RSP]
//
// Parameters:
//   - mod: Module instance implementing types.Module[M, REQ, RSP]
//   - phases: CRUD phases to register. Available phases:
//     PHASE_CREATE, PHASE_DELETE, PHASE_UPDATE, PHASE_PATCH,
//     PHASE_LIST, PHASE_GET, PHASE_CREATE_MANY, PHASE_DELETE_MANY,
//     PHASE_UPDATE_MANY, PHASE_PATCH_MANY
//
// Routes are registered based on mod.Route() and mod.Param().
// Authentication is determined by mod.Pub().
//
// Must be called during application initialization, typically in a Register() function.
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

// registerRouter registers an HTTP route with the appropriate router based on mod.Pub().
// If mod.Pub() returns true, registers with public router; otherwise with authenticated router.
func registerRouter[M types.Model, REQ types.Request, RSP types.Response](mod types.Module[M, REQ, RSP], route string, cfg *types.ControllerConfig[M], verb consts.HTTPVerb) {
	if mod.Pub() {
		// Register with public router - no authentication required
		router.Register[M, REQ, RSP](router.Pub(), route, cfg, verb)
	} else {
		// Register with authenticated router - authentication/authorization required
		router.Register[M, REQ, RSP](router.Auth(), route, cfg, verb)
	}
}
