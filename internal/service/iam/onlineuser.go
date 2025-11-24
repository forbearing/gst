package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type OnlineUserService struct {
	service.Base[*modeliam.OnlineUser, *modeliam.OnlineUser, *modeliam.OnlineUser]
}
