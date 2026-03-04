package model

import (
	"database/sql/driver"
	"strings"

	"go.uber.org/zap/zapcore"
)

type User struct {
	Name        string `json:"name,omitempty"`
	EnName      string `json:"en_name,omitempty"`
	Password    string `json:"password,omitempty"` //nolint:gosec // G117: request/model field carrying user-provided password, not a hard-coded secret
	RePassword  string `json:"re_password,omitempty" gorm:"-"`
	NewPassword string `json:"new_password,omitempty" gorm:"-"`
	Nickname    string `json:"nickname,omitempty"`

	Email          string `json:"email,omitempty"`
	EmailVerified  bool   `json:"email_verified,omitempty"`
	Mobile         string `json:"mobile,omitempty"`
	MobileVerified bool   `json:"mobile_verified,omitempty"`

	Status uint `json:"status,omitempty" gorm:"type:smallint;default:1;comment:status(0: disabled, 1: enabled)"`

	RoleID       string `json:"role_id,omitempty"`
	DepartmentID string `json:"department_id,omitempty"`

	LastLoginIP string `json:"last_login_ip,omitempty"`
	LockExpire  int64  `json:"lock_expire,omitempty"`
	LoginCount  int    `json:"login_count,omitempty"`
	NumWrong    int    `json:"num_wrong,omitempty" gorm:"comment:the number of input password wrong"`

	Avatar *Avatar `json:"avatar,omitempty"`

	LastLoginAt          *GormTime `json:"last_login,omitempty"`
	TokenExpiration      *GormTime `json:"token_expiration,omitempty"`
	LastPasswordChangeAt *GormTime `json:"last_password_change_at,omitempty" gorm:"-"`

	Token        string `json:"token,omitempty" gorm:"-"`         //nolint:gosec // G117: token value issued by external auth system, not hard-coded in source
	AccessToken  string `json:"access_token,omitempty" gorm:"-"`  //nolint:gosec // G117: token value issued by external auth system, not hard-coded in source
	RefreshToken string `json:"refresh_token,omitempty" gorm:"-"` //nolint:gosec // G117: token value issued by external auth system, not hard-coded in source
	SessionID    string `json:"session_id,omitempty" gorm:"-"`

	Base
}

type Avatar struct {
	URL    string `json:"url,omitempty"`    // 用户头像
	Thumb  string `json:"thumb,omitempty"`  // 用户头像 72x72
	Middle string `json:"middle,omitempty"` // 用户头像 240x240
	Big    string `json:"big,omitempty"`    // 用户头像 640x640
}

func (a *Avatar) Scan(value any) error        { return GormScannerWrapper(a).Scan(value) }
func (a Avatar) Value() (driver.Value, error) { return GormScannerWrapper(a).Value() }

func (u *User) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if u == nil {
		return nil
	}

	enc.AddString("name", u.Name)
	enc.AddString("email", u.Email)
	_ = enc.AddObject("base", &u.Base)

	return nil
}

// func (u *User) GetAfter() error  { return u.mask() }
// func (u *User) ListAfter() error { return u.mask() }

func (u *User) mask() error {
	u.Mobile = maskMobile(u.Mobile)
	u.Name = maskUsername(u.Name)
	u.Email = maskEmail(u.Email)
	return nil
}

// maskMobile 隐去手机号中间 4 位地区码, 如 155****8888
func maskMobile(phone string) string {
	if n := len(phone); n >= 8 {
		return phone[:n-8] + "****" + phone[n-4:]
	}
	return phone
}

// maskEmail 隐藏邮箱ID的中间部分 zhang@go-mall.com ---> z***g@go-mall.com
func maskEmail(address string) string {
	atIndex := strings.LastIndex(address, "@")
	if atIndex < 0 {
		return address
	}
	id := address[0:atIndex]
	domain := address[atIndex:]

	if len(id) <= 1 {
		return address
	}
	switch len(id) {
	case 2:
		id = id[0:1] + "*"
	case 3:
		id = id[0:1] + "*" + id[2:]
	case 4:
		id = id[0:1] + "**" + id[3:]
	default:
		masks := strings.Repeat("*", len(id)-4)
		id = id[0:2] + masks + id[len(id)-2:]
	}

	return id + domain
}

// maskUsername 保留姓名首末位 如：张三--->张* 赵丽颖--->赵*颖 欧阳娜娜--->欧**娜
func maskUsername(realName string) string {
	runeRealName := []rune(realName)
	if n := len(runeRealName); n >= 2 {
		if n == 2 {
			return string(append(runeRealName[0:1], rune('*')))
		} else {
			count := n - 2
			newRealName := runeRealName[0:1]
			for temp := 1; temp <= count; temp++ {
				newRealName = append(newRealName, rune('*'))
			}
			return string(append(newRealName, runeRealName[n-1]))
		}
	}
	return realName
}
