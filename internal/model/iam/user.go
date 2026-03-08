package modeliam

import (
	"time"

	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"golang.org/x/crypto/bcrypt"
)

// UserStatus 用户状态枚举
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"   // 激活
	UserStatusInactive UserStatus = "inactive" // 未激活
	UserStatusLocked   UserStatus = "locked"   // 锁定
)

// UserType 用户类型枚举
type UserType string

const (
	UserTypeRegular  UserType = "regular"  // 普通用户
	UserTypeAdmin    UserType = "admin"    // 管理员
	UserTypeSystem   UserType = "system"   // 系统用户
	UserTypeMerchant UserType = "merchant" // 商户用户（预留）
	UserTypeGuest    UserType = "guest"    // 访客用户
)

type UserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`

	Status           UserStatus `json:"status"`
	Type             UserType   `json:"type"`
	GroupID          string     `json:"group_id"`
	Avatar           string     `json:"avatar"`
	TwoFactorEnabled bool       `json:"two_factor_enabled"`
	IsSuperuser      bool       `json:"is_superuser"`
}

type User struct {
	Username string     `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	Status   UserStatus `json:"status" gorm:"type:varchar(20);default:'active';index"`
	Type     UserType   `json:"type" gorm:"type:varchar(20);default:'regular';index"`
	GroupID  string     `json:"group_id" gorm:"type:varchar(100);index"`
	Group    *Group     `json:"group,omitempty" gorm:"-"`

	// 个人信息
	Email       *string    `json:"email" gorm:"type:varchar(100);uniqueIndex"`
	Phone       *string    `json:"phone" gorm:"type:varchar(20);index"`
	FirstName   *string    `json:"first_name" gorm:"type:varchar(50)"`
	LastName    *string    `json:"last_name" gorm:"type:varchar(50)"`
	DisplayName *string    `json:"display_name" gorm:"type:varchar(100)"`
	Avatar      *string    `json:"avatar" gorm:"type:varchar(500)"`
	Bio         *string    `json:"bio" gorm:"type:varchar(500)"`
	Birthday    *time.Time `json:"birthday"`
	Gender      *string    `json:"gender" gorm:"type:varchar(10)"`

	// 认证信息
	Password         string `json:"password" gorm:"-"`
	PasswordHash     string `json:"-" gorm:"type:varchar(255)"`
	Salt             string `json:"-" gorm:"type:varchar(50)"`
	EmailVerified    *bool  `json:"email_verified" gorm:"default:false"`
	PhoneVerified    *bool  `json:"phone_verified" gorm:"default:false"`
	TwoFactorEnabled *bool  `json:"two_factor_enabled" gorm:"default:false"`

	// 状态管理
	IsStaff     *bool `json:"is_staff" gorm:"default:false"`
	IsSuperuser *bool `json:"is_superuser" gorm:"default:false"`

	// 多租户支持
	TenantID *string `json:"tenant_id" gorm:"index"`
	Tenant   *Tenant `json:"tenant,omitempty" gorm:"-"`

	// 登录信息
	LastLoginAt      *time.Time `json:"last_login_at"`
	LastLoginIP      *string    `json:"last_login_ip" gorm:"type:varchar(45)"`
	LoginCount       *int       `json:"login_count" gorm:"default:0"`
	FailedLoginCount int        `json:"failed_login_count" gorm:"default:0"`
	LockedUntil      *time.Time `json:"locked_until"`

	// // 验证状态
	// EmailVerificationToken  *string    `json:"-" gorm:"type:varchar(255)"`
	// EmailVerificationExpiry *time.Time `json:"-"`
	// PhoneVerificationCode   *string    `json:"-" gorm:"type:varchar(10)"`
	// PhoneVerificationExpiry *time.Time `json:"-"`
	// PasswordResetToken      *string    `json:"-" gorm:"type:varchar(255)"`
	// PasswordResetExpiry     *time.Time `json:"-"`

	// // 扩展字段
	// Preferences datatypes.JSONMap `json:"preferences"`
	// Metadata    datatypes.JSONMap `json:"metadata"`

	model.Base
}

// Design 实现 Design 方法
func (User) Design() {
	Migrate(true)
	Endpoint("users")

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
		Service(true)
	})
	Get(func() {
		Enabled(true)
		Service(true)
	})
}
func (User) Purge() bool { return true }

func (u *User) CreateBefore(ctx *types.ModelContext) error { return GenerateHashedPassword(u) }
func (u *User) UpdateBefore(ctx *types.ModelContext) error { return GenerateHashedPassword(u) }

// GenerateHashedPassword used in the scene: create user with default password.
func GenerateHashedPassword(u *User) error {
	if len(u.Password) > 0 && len(u.PasswordHash) == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.PasswordHash = string(hashedPassword)
		return nil
	}
	return nil
}
