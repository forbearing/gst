package authz

import (
	"strings"

	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/samber/lo"
)

var _ types.Module[*API, *API, APIRsp] = (*APIModule)(nil)

type APIModule struct{}

func (*APIModule) Service() types.Service[*API, *API, APIRsp] {
	return &APIService{}
}
func (*APIModule) Route() string { return "/apis" }
func (*APIModule) Pub() bool     { return false }
func (*APIModule) Param() string { return "id" }

type API struct{ model.Empty }

type APIRsp = []string

type APIService struct {
	service.Base[*API, *API, APIRsp]
}

func (APIService) List(ctx *types.ServiceContext, req *API) (APIRsp, error) {
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
