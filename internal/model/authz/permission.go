package modelauthz

import (
	"fmt"

	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"go.uber.org/zap/zapcore"
)

type Permission struct {
	Resource string `json:"resource,omitempty" schema:"resource"`
	Action   string `json:"action,omitempty" schema:"action"`

	model.Base
}

func (p *Permission) Purge() bool { return true }
func (p *Permission) CreateBefore(*types.ModelContext) error {
	p.SetID(util.HashID(p.Resource, p.Action))
	p.Remark = util.ValueOf(fmt.Sprintf("%s %s", p.Action, p.Resource))
	return nil
}

func (p *Permission) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if p == nil {
		return nil
	}
	enc.AddString("resource", p.Resource)
	enc.AddString("action", p.Action)
	_ = enc.AddObject("base", &p.Base)
	return nil
}
