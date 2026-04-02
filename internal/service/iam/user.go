package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type UserService struct {
	service.Base[*modeliam.User, *modeliam.User, *modeliam.User]
}

// DeleteAfter revokes Redis sessions for the deleted user. The controller only guarantees
// M with ID set (route/query/body id); no other fields are required.
func (UserService) DeleteAfter(_ *types.ServiceContext, u *modeliam.User) error {
	if u == nil {
		return nil
	}
	serviceiamsession.InvalidateUserSessions(u.GetID())
	return nil
}

// DeleteManyAfter revokes sessions for each deleted user. Items contain only IDs from the batch request.
func (UserService) DeleteManyAfter(_ *types.ServiceContext, users ...*modeliam.User) error {
	for _, u := range users {
		if u == nil {
			continue
		}
		serviceiamsession.InvalidateUserSessions(u.GetID())
	}
	return nil
}
