package event

import (
	"io"
	"strconv"
	"time"

	"aichat/model/example"

	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

type Lister struct {
	service.Base[*example.Event, *example.Event, *example.Event]
}

// curl http://localhost:8080/api/event?count=10&interval=300ms
func (e *Lister) List(ctx *types.ServiceContext, req *example.Event) (rsp *example.Event, err error) {
	log := e.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("event list")

	count, _ := strconv.Atoi(ctx.Query.Get("count"))
	interval, _ := time.ParseDuration(ctx.Query.Get("interval"))

	defer func() {
		_ = ctx.SSE().Done()
	}()

	i := 0
	ctx.SSE().WithInterval(interval).Stream(func(w io.Writer) bool {
		i++

		ctx.Encode(w, types.Event{
			Event: "message",
			Data:  "hello world",
		})
		return i < count
	})
	return rsp, nil
}
