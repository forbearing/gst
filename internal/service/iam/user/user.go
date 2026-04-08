package serviceiamuser

import (
	modeliamuser "github.com/forbearing/gst/internal/model/iam/user"
	serviceiamsession "github.com/forbearing/gst/internal/service/iam/session"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// UserService handles CRUD operations for IAM users.
type UserService struct {
	service.Base[*modeliamuser.User, *modeliamuser.User, *modeliamuser.User]
}

// DeleteAfter revokes Redis sessions for the deleted user. The controller only guarantees
// M with ID set (route/query/body id); no other fields are required.
func (UserService) DeleteAfter(_ *types.ServiceContext, u *modeliamuser.User) error {
	if u == nil {
		return nil
	}
	serviceiamsession.InvalidateUserSessions(u.GetID())
	return nil
}

// DeleteManyAfter revokes sessions for each deleted user. Items contain only IDs from the batch request.
func (UserService) DeleteManyAfter(_ *types.ServiceContext, users ...*modeliamuser.User) error {
	for _, u := range users {
		if u == nil {
			continue
		}
		serviceiamsession.InvalidateUserSessions(u.GetID())
	}
	return nil
}
