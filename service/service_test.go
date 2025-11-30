package service_test

import (
	"testing"

	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types/consts"
)

type TestUser struct {
	Name string

	model.Base
}

func TestRegister(t *testing.T) {
	type svc = service.Base[*TestUser, *TestUser, *TestUser]

	t.Run("pointer", func(t *testing.T) {
		service.Register[*svc](consts.PHASE_CREATE)
	})
	t.Run("struct", func(t *testing.T) {
		service.Register[svc](consts.PHASE_CREATE)
	})
}
