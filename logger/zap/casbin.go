package zap

import (
	casbinl "github.com/casbin/casbin/v2/log"
	"github.com/forbearing/gst/types"
	"go.uber.org/zap"
)

type CasbinLogger struct {
	l       types.Logger
	enabled bool
}

var _ casbinl.Logger = (*CasbinLogger)(nil)

func (c *CasbinLogger) EnableLog(enabled bool) {
	c.enabled = enabled
}

func (c *CasbinLogger) IsEnabled() bool {
	return c.enabled
}

func (c *CasbinLogger) LogModel(mode [][]string) {
	if !c.enabled {
		return
	}
	c.l.Infow("", zap.Any("mode", mode))
}

func (c *CasbinLogger) LogEnforce(matcher string, request []any, result bool, explains [][]string) {
	if !c.enabled {
		return
	}
	c.l.Infow("", zap.Bool("result", result), zap.Any("request", request), zap.Any("explains", explains), zap.String("matcher", matcher))
}

func (c *CasbinLogger) LogPolicy(policy map[string][][]string) {
	if !c.enabled {
		return
	}
	for k, vl := range policy {
		for _, v := range vl {
			c.l.Infow("policy", k, v)
		}
	}
}

func (c *CasbinLogger) LogRole(roles []string) {
	if !c.enabled {
		return
	}
	c.l.Infow("", zap.Any("roles", roles))
}

func (c *CasbinLogger) LogError(err error, msg ...string) {
	c.l.Error(err, msg)
}
