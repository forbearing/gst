package twofa

import (
	"fmt"

	"github.com/forbearing/gst/database"
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"golang.org/x/crypto/bcrypt"
)

var _ types.Module[*TOTPCheck, *TOTPCheckReq, *TOTPCheckRsp] = (*TOTPCheckModule)(nil)

type TOTPCheckModule struct{}

func (*TOTPCheckModule) Service() types.Service[*TOTPCheck, *TOTPCheckReq, *TOTPCheckRsp] {
	return &TOTPCheckService{}
}
func (*TOTPCheckModule) Route() string { return "2fa/totp/check" }
func (*TOTPCheckModule) Pub() bool     { return true }
func (*TOTPCheckModule) Param() string { return "id" }

// TOTPCheck 检查用户是否需要 2FA 验证
type TOTPCheck struct {
	model.Empty
}

func (TOTPCheck) Design() {
	Route("2fa/totp/check", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Public(true) // 公开接口，用于登录前检查
			Payload[*TOTPCheckReq]()
			Result[*TOTPCheckRsp]()
		})
	})
}

type TOTPCheckReq struct {
	Username string `json:"username" validate:"required"` // 用户名
	Password string `json:"password" validate:"required"` // 密码（用于验证权限）
}

type TOTPCheckRsp struct {
	Requires2FA bool   `json:"requires_2fa"` // 是否需要 2FA 验证
	Message     string `json:"message"`      // 响应消息
}

type TOTPCheckService struct {
	service.Base[*TOTPCheck, *TOTPCheckReq, *TOTPCheckRsp]
}

func (c *TOTPCheckService) Create(ctx *types.ServiceContext, req *TOTPCheckReq) (rsp *TOTPCheckRsp, err error) {
	log := c.WithServiceContext(ctx, ctx.GetPhase())

	// 验证输入参数
	if req.Username == "" {
		log.Warnw("empty username provided", "client_ip", ctx.ClientIP)
		return nil, fmt.Errorf("username is required")
	}
	if req.Password == "" {
		log.Warnw("empty password provided", "username", req.Username, "client_ip", ctx.ClientIP)
		return nil, fmt.Errorf("password is required")
	}

	// 查找用户
	db := database.Database[*User](ctx.DatabaseContext())
	users := make([]*User, 0)
	if err = db.WithLimit(1).WithQuery(&User{Username: req.Username}).List(&users); err != nil {
		log.Errorw("failed to query user", "username", req.Username, "error", err)
		return nil, fmt.Errorf("authentication failed")
	}
	if len(users) == 0 {
		log.Warnw("user not found", "username", req.Username, "client_ip", ctx.ClientIP)
		return nil, fmt.Errorf("authentication failed")
	}
	user := users[0]

	// 验证密码
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		log.Warnw("invalid password", "username", req.Username, "client_ip", ctx.ClientIP)
		return nil, fmt.Errorf("authentication failed")
	}

	// 检查用户是否有活跃的TOTP设备
	totpDB := database.Database[*TOTPDevice](ctx.DatabaseContext())
	devices := make([]*TOTPDevice, 0)
	if err = totpDB.WithQuery(&TOTPDevice{UserID: user.ID, IsActive: true}).List(&devices); err != nil {
		log.Errorw("failed to query TOTP devices", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to check 2FA status")
	}

	requires2FA := len(devices) > 0

	// 记录检查日志
	log.Infow("TOTP check completed",
		"username", req.Username,
		"user_id", user.ID,
		"requires_2fa", requires2FA,
		"active_devices", len(devices),
		"client_ip", ctx.ClientIP,
	)

	// 返回检查结果
	message := "2FA is not enabled"
	if requires2FA {
		message = "2FA is enabled"
	}

	return &TOTPCheckRsp{
		Requires2FA: requires2FA,
		Message:     message,
	}, nil
}

// User 是一个临时的 User 模型, 主要是为了这个接口要用到 User 模型的这两个字段, 所以在这里定义
type User struct {
	Username     string `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"type:varchar(255)"`

	model.Base
}
