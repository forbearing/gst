package twofa

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/forbearing/gst/database"
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

var _ types.Module[*TOTPConfirm, *TOTPConfirmReq, *TOTPConfirmRsp] = (*TOTPConfirmModule)(nil)

type TOTPConfirmModule struct{}

func (*TOTPConfirmModule) Service() types.Service[*TOTPConfirm, *TOTPConfirmReq, *TOTPConfirmRsp] {
	return &TOTPConfirmService{}
}
func (*TOTPConfirmModule) Route() string { return "2fa/totp/confirm" }
func (*TOTPConfirmModule) Pub() bool     { return false }
func (*TOTPConfirmModule) Param() string { return "id" }

// TOTPConfirm 确认绑定 TOTP 设备
type TOTPConfirm struct {
	model.Empty
}

func (TOTPConfirm) Design() {
	Route("2fa/totp/confirm", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Payload[*TOTPConfirmReq]()
			Result[*TOTPConfirmRsp]()
		})
	})
}

type TOTPConfirmReq struct {
	Secret     string `json:"secret" validate:"required"`     // The secret from bind step
	Code       string `json:"code" validate:"required,len=6"` // 6-digit TOTP code to confirm
	DeviceName string `json:"device_name" validate:"required,max=100"`
}

type TOTPConfirmRsp struct {
	DeviceID    string   `json:"device_id"`
	Message     string   `json:"message"`
	BackupCodes []string `json:"backup_codes"` // 8位数字备份码
}

type TOTPConfirmService struct {
	service.Base[*TOTPConfirm, *TOTPConfirmReq, *TOTPConfirmRsp]
}

func (t *TOTPConfirmService) Create(ctx *types.ServiceContext, req *TOTPConfirmReq) (rsp *TOTPConfirmRsp, err error) {
	log := t.WithServiceContext(ctx, ctx.GetPhase())

	// 1. 验证用户身份
	if len(ctx.UserID) == 0 {
		log.Errorz("user_id not found in context")
		return nil, types.NewServiceError(http.StatusUnauthorized, "authentication required")
	}

	// 2. 验证 secret 格式（Base32 编码，通常32字符）
	if len(req.Secret) == 0 {
		log.Errorz("secret is empty")
		return nil, fmt.Errorf("secret is required")
	}

	// // 验证 secret 是否为有效的 Base32 格式
	// if len(req.Secret) != 32 {
	// 	log.Errorz("invalid secret length", zap.Int("length", len(req.Secret)))
	// 	return nil, fmt.Errorf("invalid secret format")
	// }

	// 3. 验证 TOTP 代码
	valid := totp.Validate(req.Code, req.Secret)
	if !valid {
		log.Warnz("invalid totp code", zap.String("user_id", ctx.UserID))
		return nil, fmt.Errorf("invalid TOTP code")
	}

	log.Infoz("totp code validated successfully", zap.String("user_id", ctx.UserID))

	// 4. 检查是否已存在相同 secret 的设备（防止重复绑定）
	devices := make([]*TOTPDevice, 0)
	if err = database.Database[*TOTPDevice](ctx.DatabaseContext()).WithQuery(&TOTPDevice{
		UserID: ctx.UserID,
		Secret: req.Secret,
	}).WithLimit(1).List(&devices); err != nil {
		log.Errorz("failed to list devices", zap.Error(err))
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	if len(devices) > 0 {
		log.Warnz("device already exists", zap.String("user_id", ctx.UserID), zap.String("device_id", devices[0].ID))
		return nil, fmt.Errorf("device already bound")
	}

	// 5. 生成备份码
	backupCodes, err := generateBackupCodes()
	if err != nil {
		log.Errorz("failed to generate backup codes", zap.Error(err))
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// 6. 创建 TOTP 设备记录
	now := time.Now()
	device := &TOTPDevice{
		UserID:      ctx.UserID,
		DeviceName:  req.DeviceName,
		Secret:      req.Secret,
		BackupCodes: backupCodes,
		IsActive:    true,
		LastUsedAt:  &now,
	}

	if err = database.Database[*TOTPDevice](ctx.DatabaseContext()).Create(device); err != nil {
		log.Errorz("failed to create totp device", zap.Error(err))
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	log.Infoz("totp device created successfully",
		zap.String("user_id", ctx.UserID),
		zap.String("device_id", device.ID))

	// 8. 返回响应
	rsp = &TOTPConfirmRsp{
		DeviceID:    device.ID,
		Message:     "TOTP device confirmed and activated successfully",
		BackupCodes: backupCodes,
	}

	return rsp, nil
}

// generateBackupCodes 生成8个备份码，每个8位数字
func generateBackupCodes() ([]string, error) {
	codes := make([]string, 8)
	for i := range 8 {
		// 生成8位随机数字
		code := ""
		for range 8 {
			digit, err := rand.Int(rand.Reader, big.NewInt(10))
			if err != nil {
				return nil, fmt.Errorf("failed to generate random digit: %w", err)
			}
			code += digit.String()
		}
		codes[i] = code
	}
	return codes, nil
}
