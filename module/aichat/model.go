package aichat

import (
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	serviceaichat "github.com/forbearing/gst/internal/service/aichat"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*Model, *Model, *Model] = (*ModelModule)(nil)

type (
	Model        = modelaichat.Model
	ModelService = serviceaichat.ModelService
	ModelModule  struct{}
)

func (*ModelModule) Service() types.Service[*Model, *Model, *Model] {
	return &ModelService{}
}

func (*ModelModule) Route() string { return "/ai/models" }
func (*ModelModule) Pub() bool     { return false }
func (*ModelModule) Param() string { return "id" }
