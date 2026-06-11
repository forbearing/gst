package auth

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

// Login demonstrates a public action that is not backed by a database table.
type Login struct {
	model.Empty
}

// LoginRsp contains the URL a client should open to start authentication.
type LoginRsp struct {
	RedirectURL string `json:"redirect_url"`
}

func (Login) Design() {
	Route("/auth/login", func() {
		List(func() {
			Enabled(true)
			Filename("login")
			Public(true)
			Service(true)
			Result[*LoginRsp]()
		})
	})
}
