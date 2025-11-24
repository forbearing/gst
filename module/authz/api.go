package authz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	serviceauthz "github.com/forbearing/gst/internal/service/authz"
	"github.com/forbearing/gst/types"
)

var _ types.Module[*API, *API, APIRsp] = (*APIModule)(nil)

type (
	API        = modelauthz.API
	APIRsp     = modelauthz.APIRsp
	APIService = serviceauthz.APIService
	APIModule  struct{}
)

func (*APIModule) Service() types.Service[*API, *API, APIRsp] {
	return &APIService{}
}
func (*APIModule) Route() string { return "/apis" }
func (*APIModule) Pub() bool     { return false }
func (*APIModule) Param() string { return "id" }
