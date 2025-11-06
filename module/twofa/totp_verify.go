package twofa

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/forbearing/gst/database"
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

var _ types.Module[*TOTPVerify, *TOTPVerifyReq, *TOTPVerifyRsp] = (*TOTPVerifyModule)(nil)

type TOTPVerifyModule struct{}

func (*TOTPVerifyModule) Service() types.Service[*TOTPVerify, *TOTPVerifyReq, *TOTPVerifyRsp] {
	return &TOTPVerifyService{}
}
func (*TOTPVerifyModule) Route() string { return "2fa/totp/verify" }
func (*TOTPVerifyModule) Pub() bool     { return false }
func (*TOTPVerifyModule) Param() string { return "id" }

// TOTPVerify 验证 TOTP 代码
type TOTPVerify struct {
	model.Empty
}

func (TOTPVerify) Design() {
	Route("2fa/totp/verify", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Public(false) // Requires authentication
			Payload[*TOTPVerifyReq]()
			Result[*TOTPVerifyRsp]()
		})
	})
}

type TOTPVerifyReq struct {
	Code     string `json:"code" validate:"required,len=6"` // 6-digit TOTP code
	DeviceID string `json:"device_id,omitempty"`            // Optional: specific device ID
	IsBackup bool   `json:"is_backup,omitempty"`            // Whether this is a backup code
}

type TOTPVerifyRsp struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
}

type TOTPVerifyService struct {
	service.Base[*TOTPVerify, *TOTPVerifyReq, *TOTPVerifyRsp]
}

func (t *TOTPVerifyService) Create(ctx *types.ServiceContext, req *TOTPVerifyReq) (rsp *TOTPVerifyRsp, err error) {
	log := t.WithServiceContext(ctx, ctx.GetPhase())

	// 1. 验证用户身份
	if len(ctx.UserID) == 0 {
		log.Errorz("user_id not found in context")
		return &TOTPVerifyRsp{
			Valid:   false,
			Message: "authentication required",
		}, types.NewServiceError(http.StatusUnauthorized, "authentication required")
	}

	// 2. 验证输入参数
	if len(req.Code) == 0 {
		log.Errorz("code is empty")
		return &TOTPVerifyRsp{
			Valid:   false,
			Message: "verification code is required",
		}, fmt.Errorf("verification code is required")
	}

	// 3. 查询用户的 TOTP 设备
	devices := make([]*TOTPDevice, 0)
	query := &TOTPDevice{
		UserID:   ctx.UserID,
		IsActive: true,
	}

	// 如果指定了设备ID，则只查询该设备
	if len(req.DeviceID) > 0 {
		query.Base.ID = req.DeviceID
	}

	if err = database.Database[*TOTPDevice](ctx.DatabaseContext()).WithQuery(query).List(&devices); err != nil {
		log.Errorz("failed to list totp devices", zap.Error(err))
		return &TOTPVerifyRsp{
			Valid:   false,
			Message: "failed to retrieve device information",
		}, fmt.Errorf("failed to list devices: %w", err)
	}

	if len(devices) == 0 {
		log.Warnz("no active totp devices found", zap.String("user_id", ctx.UserID))
		return &TOTPVerifyRsp{
			Valid:   false,
			Message: "no active TOTP devices found",
		}, fmt.Errorf("no active TOTP devices found")
	}

	// 4. 验证代码
	var validDevice *TOTPDevice
	var isBackupCodeUsed bool

	for _, device := range devices {
		if req.IsBackup {
			// 验证备份码
			if t.validateBackupCode(req.Code, device) {
				validDevice = device
				isBackupCodeUsed = true
				break
			}
		} else {
			// 验证 TOTP 代码
			if totp.Validate(req.Code, device.Secret) {
				validDevice = device
				break
			}
		}
	}

	if validDevice == nil {
		log.Warnz("invalid verification code",
			zap.String("user_id", ctx.UserID),
			zap.Bool("is_backup", req.IsBackup))
		return &TOTPVerifyRsp{
			Valid:   false,
			Message: "invalid verification code",
		}, nil
	}

	// 5. 更新设备状态
	now := time.Now()
	validDevice.LastUsedAt = &now

	// 如果使用了备份码，从列表中移除
	if isBackupCodeUsed {
		t.removeUsedBackupCode(req.Code, validDevice)
	}

	// 保存设备更新
	if err = database.Database[*TOTPDevice](ctx.DatabaseContext()).Update(validDevice); err != nil {
		log.Errorz("failed to update device", zap.Error(err))
		// 即使更新失败，验证仍然成功，只记录错误
		log.Warnz("device update failed but verification succeeded")
	}

	log.Infoz("totp verification successful",
		zap.String("user_id", ctx.UserID),
		zap.String("device_id", validDevice.ID),
		zap.Bool("is_backup", req.IsBackup))

	return &TOTPVerifyRsp{
		Valid:   true,
		Message: "verification successful",
	}, nil
}

// validateBackupCode 验证备份码
func (t *TOTPVerifyService) validateBackupCode(code string, device *TOTPDevice) bool {
	// 备份码应该是8位数字
	if len(code) != 8 {
		return false
	}

	// 检查是否在备份码列表中
	return slices.Contains(device.BackupCodes, code)
}

// removeUsedBackupCode 从备份码列表中移除已使用的码
func (t *TOTPVerifyService) removeUsedBackupCode(code string, device *TOTPDevice) {
	for i, backupCode := range device.BackupCodes {
		if backupCode == code {
			// 移除已使用的备份码
			device.BackupCodes = slices.Delete(device.BackupCodes, i, i+1)
			break
		}
	}
}
