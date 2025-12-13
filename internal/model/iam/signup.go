package modeliam

import (
	"github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
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
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
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
