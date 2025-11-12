package modelauthz

import (
	"strings"

	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"go.uber.org/zap/zapcore"
	"gorm.io/datatypes"
)

var (
	RootID      = model.RootID
	RootName    = model.RootName
	UnknownID   = model.UnknownID
	UnknownName = model.UnknownName
	NoneID      = model.NoneID
	NoneName    = model.NoneName

	KeyName = model.KeyName
	KeyID   = model.KeyID
)

var MenuRoot = &Menu{ParentID: model.RootID, Base: model.Base{ID: RootID}}

type MenuPlatform string

const (
	MenuPlatformAll     = "all"
	MenuPlatformWeb     = "web"
	MenuPlatformMobile  = "mobile"
	MenuPlatformDesktop = "desktop"
)

type Menu struct {
	API     datatypes.JSONSlice[string] `json:"api,omitempty" schema:"api"`         // 后端路由, 如果为空则使用 "/api" + Path
	Path    string                      `json:"path,omitempty" schema:"path"`       // path should not add `omitempty` tag, empty value means default router in react route6.x.
	Element string                      `json:"element,omitempty" schema:"element"` // 前端页面组件
	Label   string                      `json:"label,omitempty" schema:"label"`     // 页面组件左侧的菜单名
	Icon    string                      `json:"icon,omitempty" schema:"icon"`       // 页面组件左侧的菜单图标

	Visiable *bool  `json:"visiable,omitempty" schema:"visiable" gorm:"default:1"`                                                   // 前端页面路由是否可见
	Default  string `json:"default,omitempty" schema:"default"`                                                                      // 子路由中的默认路由, 如果有 Children, Default 才可能存在
	Status   *uint  `json:"status,omitempty" gorm:"type:smallint;default:1;comment:status(0: disabled, 1: enabled)" schema:"status"` // 该路由是否启用

	ParentID string  `json:"parent_id,omitempty" gorm:"size:191" schema:"parent_id"`
	Children []*Menu `json:"children,omitempty" gorm:"foreignKey:ParentID"`             // 子路由
	Parent   *Menu   `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID"` // 父路由

	// the empty value of `Platform` means all.
	Platform MenuPlatform `json:"platform,omitempty" schema:"platform"`

	DomainPattern string `json:"domain_pattern,omitempty" schema:"domain_pattern" gorm:"default:.*"`

	model.Base
}

func (m *Menu) CreateBefore(ctx *types.ModelContext) (err error) { return m.validate() }
func (m *Menu) UpdateBefore(ctx *types.ModelContext) error       { return m.validate() }

func (m *Menu) Expands() []string { return []string{"Children", "Parent"} }
func (m *Menu) Excludes() map[string][]any {
	return map[string][]any{KeyID: {RootID, UnknownID, NoneID}}
}

// // ListAfter 可能是只查询最顶层的 Menu,并不能拿到最顶层的 Menu
// func (m *Menu) ListAfter(ctx *types.ModelContext) (err error) {
// 	oldPath, oldAPI := m.Path, m.API
// 	if err = m.checkPathAndAPI(); err != nil {
// 		return err
// 	}
// 	if m.Path != oldPath || m.API != oldAPI {
// 		return database.Database[*Menu](ctx.DatabaseContext()).WithoutHook().Update(m)
// 	}
// 	return nil
// }

// func (m *Menu) GetAfter(ctx *types.ModelContext) (err error) {
// 	oldPath, oldAPI := m.Path, m.API
// 	if err = m.checkPathAndAPI(); err != nil {
// 		return err
// 	}
// 	if m.Path != oldPath || m.API != oldAPI {
// 		return database.Database[*Menu](ctx.DatabaseContext()).WithoutHook().Update(m)
// 	}
// 	return nil
// }

func (m *Menu) validate() error {
	if len(m.ParentID) == 0 {
		m.ParentID = RootID
	}
	if m.Visiable == nil {
		m.Visiable = util.ValueOf(true)
	}
	if len(m.DomainPattern) == 0 {
		m.DomainPattern = ".*"
	}
	if len(m.Path) > 0 {
		m.Path = strings.TrimSuffix(strings.TrimSpace(m.Path), "/")
	}
	return nil
}

// func (m *Menu) checkPathAndAPI() (err error) {
// 	// 去除空格和尾部所有的 /
// 	m.Path = strings.TrimSpace(m.Path)
// 	m.Path = strings.TrimRight(m.Path, "/")
//
// 	// 检查是否是有效的 url
// 	var newPath string
// 	if newPath, err = url.JoinPath("/", m.Path); err != nil {
// 		return err
// 	}
//
// 	// m.Path 可能为空,如果这是一个父级菜单的话,则Path为空
// 	if len(m.Path) == 0 {
// 		m.API = ""
// 	}
// 	if len(m.Path) > 0 && len(m.API) == 0 {
// 		if m.API, err = url.JoinPath("/api", m.Path); err != nil {
// 			return err
// 		}
// 	}
//
// 	// 有一些 path 不是以 / 开头的, 我们需要手动加上
// 	if len(m.Path) > 0 {
// 		m.Path = newPath
// 	}
//
// 	return nil
// }

func (m *Menu) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if m == nil {
		return nil
	}
	enc.AddString("api", strings.Join(m.API, ","))
	enc.AddString("path", m.Path)
	enc.AddString("label", m.Label)
	enc.AddString("element", m.Element)
	enc.AddInt("children len", len(m.Children))

	return nil
}
