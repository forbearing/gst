package column

import (
	"strings"

	"github.com/forbearing/gst/types"
)

func (s *srv) Get(ctx *types.ServiceContext, req *empty) (rsp, error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	table := strings.ReplaceAll(ctx.Params["id"], "-", "_")
	columns, ok := tableColumns[table]
	if !ok {
		log.Warnw("not register table", "table", table)
	}

	return new(column).QueryColumns(ctx.Query, table, columns)
}
