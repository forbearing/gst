package middleware

import (
	"strings"
	"sync"

	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
)

// RouteParams is a middleware to get route parameters
func RouteParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(consts.PARAMS, RouteManager.Get(c.FullPath()))
		c.Next()
	}
}

type routeParamsManager struct {
	paramsMap map[string][]string
	mu        sync.RWMutex
}

func NewRouteParamsManager() *routeParamsManager {
	return &routeParamsManager{
		paramsMap: make(map[string][]string),
	}
}

func (rpm *routeParamsManager) Add(path string) {
	path = strings.TrimSpace(path)
	if len(path) == 0 {
		return
	}
	rpm.mu.Lock()
	rpm.paramsMap[path] = rpm.parsePath(path)
	rpm.mu.Unlock()
}

func (rpm *routeParamsManager) Get(path string) []string {
	rpm.mu.RLock()
	defer rpm.mu.RUnlock()
	val := rpm.paramsMap[path]
	if len(val) == 0 {
		// NOTE: {}string <nil> not deep equal to []string{}
		// map[key] returns {}string <nil> not []string{}
		return []string{}
	}
	return val
}

func (rpm *routeParamsManager) parsePath(path string) []string {
	parts := strings.Split(path, "/")
	var params []string

	for _, part := range parts {
		if after, ok := strings.CutPrefix(part, ":"); ok {
			param := after
			if len(param) > 0 {
				params = append(params, param)
			}
		} else if strings.Contains(part, "{") && strings.Contains(part, "}") {
			// 处理 {id} 风格的参数 (如果需要)
			param := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			if len(param) > 0 {
				params = append(params, param)
			}
		}
	}

	return params
}
