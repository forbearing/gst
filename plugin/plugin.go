package plugin

import (
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
)

func Use[M types.Model, REQ types.Request, RSP types.Response, S types.Service[M, REQ, RSP]](plugin types.Plugin[M, REQ, RSP], phase ...consts.Phase) {
	model.Register[M]()
	for _, p := range phase {
		service.Register[S](p)
	}
}
