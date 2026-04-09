package serviceiamgroup

import (
	modeliamgroup "github.com/forbearing/gst/internal/model/iam/group"
	"github.com/forbearing/gst/service"
)

// GroupService handles CRUD operations for IAM groups.
type GroupService struct {
	service.Base[*modeliamgroup.Group, *modeliamgroup.Group, *modeliamgroup.Group]
}
