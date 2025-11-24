package serviceauthz

import (
	"github.com/forbearing/gst/database"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/samber/lo"
)

type APIService struct {
	service.Base[*modelauthz.API, *modelauthz.API, modelauthz.APIRsp]
}

func (APIService) List(ctx *types.ServiceContext, req *modelauthz.API) (modelauthz.APIRsp, error) {
	perms := make([]*modelauthz.Permission, 0)
	if err := database.Database[*modelauthz.Permission](ctx.DatabaseContext()).List(&perms); err != nil {
		return nil, err
	}

	apis := make([]string, 0)
	for _, pem := range perms {
		// api := strings.TrimSuffix(pem.Resource, "/{id}")
		// api = strings.TrimSuffix(api, "/id")
		// api = strings.TrimSuffix(api, "/batch")
		// api = strings.TrimSuffix(api, "/")
		// apis = append(apis, api)
		apis = append(apis, pem.Resource)
	}

	return lo.Uniq(apis), nil
}
