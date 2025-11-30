package aichat

import (
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	serviceaichat "github.com/forbearing/gst/internal/service/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types/consts"
)

type (
	Model    = modelaichat.Model
	Provider = modelaichat.Provider
)

func Register() {
	// Register "Model" module.
	module.Use[
		*Model,
		*Model,
		*Model,
		*service.Base[*Model, *Model, *Model]](
		module.NewWrapper[*Model, *Model, *Model]("/api/models", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "Provider" module.
	module.Use[
		*Provider,
		*Provider,
		*Provider,
		*service.Base[*Provider, *Provider, *Provider]](
		module.NewWrapper[*Provider, *Provider, *Provider]("/api/providers", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "TestConnection" module.
	// Route: POST /api/ai/providers/test-connection
	// Request body: Provider (with config information)
	module.Use[
		*model.Empty,
		*modelaichat.Provider,
		*modelaichat.TestConnectionRsp,
		*serviceaichat.TestConnection](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.TestConnectionRsp](
			"/ai/providers/test-connection",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "ListModels" module.
	// Route: POST /api/ai/providers/models
	// Request body: Provider (with config information)
	module.Use[
		*model.Empty,
		*modelaichat.Provider,
		*modelaichat.ListModelsRsp,
		*serviceaichat.ListModels](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.ListModelsRsp](
			"/ai/providers/models",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)
}
