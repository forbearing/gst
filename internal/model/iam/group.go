package modeliam

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
)

var DefaultGroup = Group{
	Name: "default",
	Base: model.Base{ID: "default"},
}

// GroupType 用户组类型枚举
type GroupType string

const (
	GroupTypeRegular    GroupType = "regular"
	GroupTypeDepartment GroupType = "department"
	GroupTypeTeam       GroupType = "team"
	GroupTypeProject    GroupType = "project"
	GroupTypeRole       GroupType = "role"
)

// GroupStatus 用户组状态枚举
type GroupStatus string

const (
	GroupStatusActive   GroupStatus = "active"   // 激活
	GroupStatusInactive GroupStatus = "inactive" // 未激活
)

// Group 用户组模型
type Group struct {
	// 基本信息
	Name   string      `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	Type   GroupType   `json:"type" gorm:"type:varchar(20);default:'regular';index"`
	Status GroupStatus `json:"status" gorm:"type:varchar(20);default:'active';index"`

	// 层级关系
	ParentID *string `json:"parent_id" gorm:"index"`
	Path     string  `json:"path" gorm:"type:varchar(500);index"` // 层级路径，如 /1/2/3
	Level    int     `json:"level" gorm:"default:0;index"`        // 层级深度

	// 多租户支持
	TenantID *string `json:"tenant_id" gorm:"index"`
	Tenant   *Tenant `json:"tenant,omitempty" gorm:"-"`

	// // 组织信息
	// ManagerID   *string `json:"manager_id" gorm:"index"`               // 组长/管理员ID
	// ManagerName *string `json:"manager_name" gorm:"type:varchar(100)"` // 管理员姓名
	// Email       *string `json:"email" gorm:"type:varchar(100)"`        // 组邮箱
	// Phone       *string `json:"phone" gorm:"type:varchar(20)"`         // 组电话
	// Location    *string `json:"location" gorm:"type:varchar(200)"`     // 办公地点
	// CostCenter  *string `json:"cost_center" gorm:"type:varchar(50)"`   // 成本中心

	// // 统计信息
	// MemberCount       int `json:"member_count" gorm:"-"`        // 成员数量
	// DirectMemberCount int `json:"direct_member_count" gorm:"-"` // 直接成员数量
	// SubGroupCount     int `json:"sub_group_count" gorm:"-"`     // 子组数量

	// // 扩展字段
	// Settings datatypes.JSONMap `json:"settings"`
	// Metadata datatypes.JSONMap `json:"metadata"`

	model.Base
}

func (Group) Design() {
	Migrate(true)
	Endpoint("groups")

	Create(func() {
		Enabled(true)
	})
	Delete(func() {
		Enabled(true)
	})
	Update(func() {
		Enabled(true)
	})
	Patch(func() {
		Enabled(true)
	})
	List(func() {
		Enabled(true)
	})
	Get(func() {
		Enabled(true)
	})
}

func (Group) Purge() bool { return true }
