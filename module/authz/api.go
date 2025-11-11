package authz

import (
	"strings"

	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/samber/lo"
)

var _ types.Module[*Api, *Api, ApiRsp] = (*ApiModule)(nil)

type ApiModule struct{}

func (*ApiModule) Service() types.Service[*Api, *Api, ApiRsp] {
	return &ApiService{}
}
func (*ApiModule) Route() string { return "/apis" }
func (*ApiModule) Pub() bool     { return false }
func (*ApiModule) Param() string { return "id" }

type Api struct{ model.Empty }

type ApiRsp = []string

type ApiService struct {
	service.Base[*Api, *Api, ApiRsp]
}

func (ApiService) List(ctx *types.ServiceContext, req *Api) (ApiRsp, error) {
	perms := make([]*Permission, 0)
	if err := database.Database[*Permission](ctx.DatabaseContext()).List(&perms); err != nil {
		return nil, err
	}

	apis := make([]string, 0)
	for _, pem := range perms {
		api := strings.TrimRight(pem.Resource, "/{id}")
		api = strings.TrimSuffix(api, "/id")
		api = strings.TrimSuffix(api, "/batch")
		api = strings.TrimSuffix(api, "/")
		apis = append(apis, api)
	}

	return lo.Uniq(apis), nil
}
