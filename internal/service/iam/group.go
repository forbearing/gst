package serviceiam

import (
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
)

type GroupService struct {
	service.Base[*modeliam.Group, *modeliam.Group, *modeliam.Group]
}
