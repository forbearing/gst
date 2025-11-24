package modeliam

import (
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
)

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusExpired   TenantStatus = "expired"
)

// TenantType 租户类型枚举
type TenantType string

const (
	TenantTypeEnterprise TenantType = "enterprise"
	TenantTypePro        TenantType = "pro"
	TenantTypeBasic      TenantType = "basic"
	TenantTypeTrial      TenantType = "trial"
)

// Tenant 租户模型（预留多租户功能）
type Tenant struct {
	// 基本信息
	Name   string       `json:"name" gorm:"type:varchar(100);not null"`
	Status TenantStatus `json:"status" gorm:"type:varchar(20);default:'inactive';index"`
	Type   TenantType   `json:"type" gorm:"type:varchar(20);default:'basic';index"`

	// // 联系信息
	// Email      *string `json:"email" gorm:"type:varchar(100)"`
	// Phone      *string `json:"phone" gorm:"type:varchar(20)"`
	// Website    *string `json:"website" gorm:"type:varchar(200)"`
	// Address    *string `json:"address" gorm:"type:varchar(500)"`
	// Country    *string `json:"country" gorm:"type:varchar(50)"`
	// City       *string `json:"city" gorm:"type:varchar(50)"`
	// PostalCode *string `json:"postal_code" gorm:"type:varchar(20)"`

	// // 管理信息
	// AdminUserID   *uint   `json:"admin_user_id" gorm:"index"`             // 管理员用户ID
	// AdminUsername *string `json:"admin_username" gorm:"type:varchar(50)"` // 管理员用户名
	// AdminEmail    *string `json:"admin_email" gorm:"type:varchar(100)"`   // 管理员邮箱

	// // 订阅和计费信息
	// PlanID       *uint      `json:"plan_id" gorm:"index"`                  // 订阅计划ID
	// PlanName     *string    `json:"plan_name" gorm:"type:varchar(50)"`     // 计划名称
	// SubscribedAt *time.Time `json:"subscribed_at"`                         // 订阅时间
	// ExpiresAt    *time.Time `json:"expires_at"`                            // 过期时间
	// TrialEndsAt  *time.Time `json:"trial_ends_at"`                         // 试用结束时间
	// BillingCycle string     `json:"billing_cycle" gorm:"type:varchar(20)"` // 计费周期：monthly, yearly

	// // 配额和限制
	// MaxUsers       int `json:"max_users" gorm:"default:10"`          // 最大用户数
	// MaxGroups      int `json:"max_groups" gorm:"default:5"`          // 最大组数
	// MaxStorage     int `json:"max_storage" gorm:"default:1024"`      // 最大存储空间(MB)
	// MaxProjects    int `json:"max_projects" gorm:"default:3"`        // 最大项目数
	// MaxAPIRequests int `json:"max_api_requests" gorm:"default:1000"` // 每日API请求限制

	// // 当前使用情况
	// CurrentUsers    int `json:"current_users" gorm:"default:0"`    // 当前用户数
	// CurrentGroups   int `json:"current_groups" gorm:"default:0"`   // 当前组数
	// CurrentStorage  int `json:"current_storage" gorm:"default:0"`  // 当前存储使用量(MB)
	// CurrentProjects int `json:"current_projects" gorm:"default:0"` // 当前项目数

	// // 功能开关, 自定义配置
	// Features datatypes.JSONMap `json:"features"` // 功能开关配置
	// Settings datatypes.JSONMap `json:"settings"` // 租户设置
	// Metadata datatypes.JSONMap `json:"metadata"` // 元数据

	// // 品牌定制
	// Logo      *string `json:"logo" gorm:"type:varchar(500)"`   // Logo URL
	// Theme     *string `json:"theme" gorm:"type:varchar(50)"`   // 主题配置
	// CustomCSS *string `json:"custom_css" gorm:"type:text"`     // 自定义CSS
	// Domain    *string `json:"domain" gorm:"type:varchar(100)"` // 自定义域名

	model.Base
}

func (Tenant) Design() {
	Migrate(false)
	Enabled(true)
	Endpoint("tenants")

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

func (Tenant) Purge() bool { return true }

// // CreateBefore 创建前的钩子函数
// func (t *Tenant) CreateBefore(ctx *types.ModelContext) error {
// 	if t.Status == "" {
// 		t.Status = TenantStatusActive
// 	}
// 	if t.Type == "" {
// 		t.Type = TenantTypeBasic
// 	}
//
// 	// 设置默认配额
// 	if t.MaxUsers == 0 {
// 		t.MaxUsers = 10
// 	}
// 	if t.MaxGroups == 0 {
// 		t.MaxGroups = 5
// 	}
// 	if t.MaxStorage == 0 {
// 		t.MaxStorage = 1024 // 1GB
// 	}
// 	if t.MaxProjects == 0 {
// 		t.MaxProjects = 3
// 	}
// 	if t.MaxAPIRequests == 0 {
// 		t.MaxAPIRequests = 1000
// 	}
//
// 	return nil
// }

type TenantService struct {
	service.Base[*Tenant, *Tenant, *Tenant]
}
