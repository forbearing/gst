package dao

import (
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/types"
)

// QueryModelMap 从数据库中查询类型为 M 数据, 返回 key 为 keyFunc(M) 的 map
func QueryModelMap[M types.Model](ctx *types.DatabaseContext, keyFunc func(M) string, queryFunc func() M) (map[string]M, error) {
	if keyFunc == nil {
		keyFunc = func(m M) string {
			return m.GetID()
		}
	}
	if queryFunc == nil {
		queryFunc = func() M {
			var m M
			return m
		}
	}

	objs := make([]M, 0)
	if err := database.Database[M](ctx).
		WithQuery(queryFunc(), types.QueryConfig{AllowEmpty: true}).
		List(&objs); err != nil {
		return nil, err
	}

	objMap := make(map[string]M)
	for _, obj := range objs {
		objMap[keyFunc(obj)] = obj
	}

	return objMap, nil
}
