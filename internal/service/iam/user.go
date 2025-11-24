package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type UserService struct {
	service.Base[*modeliam.User, *modeliam.User, *modeliam.User]
}
