package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type Signup struct {
	model.Empty
}

type SignupReq struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RePassword string `json:"re_password"`
	Email      string `json:"email,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty"`
}

type SignupRsp struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

func (Signup) Design() {
	dsl.Create(func() {
		dsl.Enabled(true)
		dsl.Public(true)
		dsl.Service(true)
		dsl.Payload[*SignupReq]()
		dsl.Result[*SignupRsp]()
	})
}

type SignupService struct {
	service.Base[*Signup, *SignupReq, *SignupRsp]
}
