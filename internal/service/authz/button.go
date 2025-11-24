package serviceauthz

import (
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/service"
)

type ButtonService struct {
	service.Base[*modelauthz.Button, *modelauthz.Button, *modelauthz.Button]
}
