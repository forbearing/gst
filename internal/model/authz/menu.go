package modelauthz

import (
	"net/url"
	"strings"

	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
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

// Menu: 菜单
// TODO: 加一个 api 用来指定后端路由,如果为空则使用 Path.
type Menu struct {
	API     string `json:"api,omitempty" schema:"api"` // 后端路由, 如果为空则使用 "/api" + Path
	Path    string `json:"path" schema:"path"`         // path should not add `omitempty` tag, empty value means default router in react route6.x.
	Element string `json:"element,omitempty" schema:"element"`
	Label   string `json:"label,omitempty" schema:"label"`
	Icon    string `json:"icon,omitempty" schema:"icon"`

	Visiable *bool  `json:"visiable" schema:"visiable"`                                                                    // 默认路由
	Default  string `json:"default,omitempty" schema:"default"`                                                            // 自路由中的默认路由, 如果有 Children, Default 才可能存在
	Status   *uint  `json:"status" gorm:"type:smallint;default:1;comment:status(0: disabled, 1: enabled)" schema:"status"` // 该路由是否启用

	// RoleIds GormStrings `json:"role_ids,omitempty"`
	// Roles   []*Role     `json:"roles,omitempty" gorm:"-"`

	ParentID string  `json:"parent_id,omitempty" gorm:"size:191" schema:"parent_id"`
	Children []*Menu `json:"children,omitempty" gorm:"foreignKey:ParentID"`             // 子路由
	Parent   *Menu   `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID"` // 父路由

	// the empty value of `Platform` means all.
	Platform MenuPlatform `json:"platform" schema:"platform"`

	DomainPattern string `json:"domain_pattern" schema:"domain_pattern"`

	model.Base
}

func (m *Menu) Expands() []string { return []string{"Children", "Parent"} }
func (m *Menu) Excludes() map[string][]any {
	return map[string][]any{KeyID: {RootID, UnknownID, NoneID}}
}

func (m *Menu) CreateBefore(ctx *types.ModelContext) (err error) {
	return multierr.Combine(m.initDefaultValue(), m.checkPathAndAPI())
}

func (m *Menu) UpdateBefore(ctx *types.ModelContext) error {
	return multierr.Combine(m.initDefaultValue(), m.checkPathAndAPI())
}

// ListAfter 可能是只查询最顶层的 Menu,并不能拿到最顶层的 Menu
func (m *Menu) ListAfter(ctx *types.ModelContext) (err error) {
	oldPath, oldAPI := m.Path, m.API
	if err = m.checkPathAndAPI(); err != nil {
		return err
	}
	if m.Path != oldPath || m.API != oldAPI {
		return database.Database[*Menu](ctx.DatabaseContext()).WithoutHook().Update(m)
	}
	return nil
}

func (m *Menu) GetAfter(ctx *types.ModelContext) (err error) {
	oldPath, oldAPI := m.Path, m.API
	if err = m.checkPathAndAPI(); err != nil {
		return err
	}
	if m.Path != oldPath || m.API != oldAPI {
		return database.Database[*Menu](ctx.DatabaseContext()).WithoutHook().Update(m)
	}
	return nil
}

func (m *Menu) initDefaultValue() error {
	if len(m.ParentID) == 0 {
		m.ParentID = RootID
	}
	if m.Visiable == nil {
		m.Visiable = util.ValueOf(true)
	}
	if len(m.DomainPattern) == 0 {
		m.DomainPattern = ".*"
	}
	return nil
}

func (m *Menu) checkPathAndAPI() (err error) {
	// 去除空格和尾部所有的 /
	m.Path = strings.TrimSpace(m.Path)
	m.Path = strings.TrimRight(m.Path, "/")

	// 检查是否是有效的 url
	var newPath string
	if newPath, err = url.JoinPath("/", m.Path); err != nil {
		return err
	}

	// m.Path 可能为空,如果这是一个父级菜单的话,则Path为空
	if len(m.Path) == 0 {
		m.API = ""
	}
	if len(m.Path) > 0 && len(m.API) == 0 {
		if m.API, err = url.JoinPath("/api", m.Path); err != nil {
			return err
		}
	}

	// 有一些 path 不是以 / 开头的, 我们需要手动加上
	if len(m.Path) > 0 {
		m.Path = newPath
	}

	return nil
}

func (m *Menu) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if m == nil {
		return nil
	}
	enc.AddString("api", m.API)
	enc.AddString("path", m.Path)
	enc.AddString("label", m.Label)
	enc.AddString("element", m.Element)
	enc.AddInt("children len", len(m.Children))

	return nil
}
