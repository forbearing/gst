package modeliam

import (
	"time"

	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"golang.org/x/crypto/bcrypt"
)

// UserStatus is the account lifecycle / access state for IAM users.
// Use UserStatusInactive for administrative disable (no login; revoke existing sessions via service hooks).
// Use UserStatusLocked for security lockout (same login denial; semantics are policy-specific).
// UserStatusActive is the normal operating state.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

// UserType defines IAM user categories.
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
	Username string `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	// Status: "active" (default) allows login; "inactive" disables login (administrative); "locked" denies login (lockout).
	Status  UserStatus `json:"status" gorm:"type:varchar(20);default:'active';index"`
	Type    UserType   `json:"type" gorm:"type:varchar(20);default:'regular';index"`
	GroupID string     `json:"group_id" gorm:"type:varchar(100);index"`
	Group   *Group     `json:"group,omitempty" gorm:"-"`

	// Profile
	Email       *string    `json:"email" gorm:"type:varchar(100);uniqueIndex"`
	Phone       *string    `json:"phone" gorm:"type:varchar(20);index"`
	FirstName   *string    `json:"first_name" gorm:"type:varchar(50)"`
	LastName    *string    `json:"last_name" gorm:"type:varchar(50)"`
	DisplayName *string    `json:"display_name" gorm:"type:varchar(100)"`
	Avatar      *string    `json:"avatar" gorm:"type:varchar(500)"`
	Bio         *string    `json:"bio" gorm:"type:varchar(500)"`
	Birthday    *time.Time `json:"birthday"`
	Gender      *string    `json:"gender" gorm:"type:varchar(10)"`

	// Credentials and auth settings
	Password         string `json:"password" gorm:"-"`
	PasswordHash     string `json:"-" gorm:"type:varchar(255)"`
	Salt             string `json:"-" gorm:"type:varchar(50)"`
	TwoFactorEnabled *bool  `json:"two_factor_enabled" gorm:"default:false"`
	// MustChangePassword is set when an administrator resets the password; the user must change it before other protected APIs are allowed.
	MustChangePassword bool `json:"must_change_password" gorm:"default:false;not null"`

	// Verification and email lifecycle
	EmailVerified      *bool      `json:"email_verified" gorm:"default:false"`
	EmailVerifiedAt    *time.Time `json:"email_verified_at"`
	PhoneVerified      *bool      `json:"phone_verified" gorm:"default:false"`
	LastEmailChangedAt *time.Time `json:"last_email_changed_at"`

	// Authorization flags
	IsStaff     *bool `json:"is_staff" gorm:"default:false"`
	IsSuperuser *bool `json:"is_superuser" gorm:"default:false"`

	// Multi-tenant scope
	TenantID *string `json:"tenant_id" gorm:"index"`
	Tenant   *Tenant `json:"tenant,omitempty" gorm:"-"`

	// Login activity
	LastLoginAt      *time.Time `json:"last_login_at"`
	LastLoginIP      *string    `json:"last_login_ip" gorm:"type:varchar(45)"`
	LoginCount       *int       `json:"login_count" gorm:"default:0"`
	FailedLoginCount int        `json:"failed_login_count" gorm:"default:0"`
	LockedUntil      *time.Time `json:"locked_until"`

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
