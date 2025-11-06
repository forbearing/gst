package twofa

import (
	"fmt"
	"net/http"

	"github.com/forbearing/gst/database"
	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"go.uber.org/zap"
)

var _ types.Module[*TOTPStatus, *TOTPStatus, *TOTPStatusRsp] = (*TOTPStatusModule)(nil)

type TOTPStatusModule struct{}

func (*TOTPStatusModule) Service() types.Service[*TOTPStatus, *TOTPStatus, *TOTPStatusRsp] {
	return &TOTPStatusService{}
}
func (*TOTPStatusModule) Route() string { return "2fa/totp/status" }
func (*TOTPStatusModule) Pub() bool     { return false }
func (*TOTPStatusModule) Param() string { return "id" }

// TOTPStatus 获取用户的 2FA 状态
type TOTPStatus struct {
	model.Empty
}

func (TOTPStatus) Design() {
	Route("2fa/totp/status", func() {
		List(func() {
			Enabled(true)
			Service(true)
			Result[*TOTPStatusRsp]()
		})
	})
}

type TOTPStatusRsp struct {
	Enabled     bool             `json:"enabled"`      // Whether 2FA is enabled
	DeviceCount int              `json:"device_count"` // Number of active devices
	Devices     []TOTPDeviceInfo `json:"devices"`      // List of devices (without secrets)
}

type TOTPDeviceInfo struct {
	ID         string  `json:"id"`
	DeviceName string  `json:"device_name"`
	IsActive   bool    `json:"is_active"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type TOTPStatusService struct {
	service.Base[*TOTPStatus, *TOTPStatus, *TOTPStatusRsp]
}

func (t *TOTPStatusService) List(ctx *types.ServiceContext, req *TOTPStatus) (rsp *TOTPStatusRsp, err error) {
	log := t.WithServiceContext(ctx, ctx.GetPhase())

	// 1. 验证用户身份
	if len(ctx.UserID) == 0 {
		log.Errorz("user_id not found in context")
		return nil, types.NewServiceError(http.StatusUnauthorized, "authentication required")
	}

	// 2. 查询用户的所有 TOTP 设备
	devices := make([]*TOTPDevice, 0)
	query := &TOTPDevice{
		UserID: ctx.UserID,
	}

	if err = database.Database[*TOTPDevice](ctx.DatabaseContext()).WithQuery(query).List(&devices); err != nil {
		log.Errorz("failed to list totp devices", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve device information: %w", err)
	}

	// 3. 统计设备状态和转换设备信息
	activeDeviceCount := 0
	deviceInfos := make([]TOTPDeviceInfo, 0, len(devices))

	for _, device := range devices {
		// 统计活跃设备数量
		if device.IsActive {
			activeDeviceCount++
		}

		// 转换设备信息（不包含敏感信息）
		deviceInfo := TOTPDeviceInfo{
			ID:         device.ID,
			DeviceName: device.DeviceName,
			IsActive:   device.IsActive,
			CreatedAt:  device.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), // RFC3339 格式
		}

		// 格式化最后使用时间
		if device.LastUsedAt != nil {
			lastUsedStr := device.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
			deviceInfo.LastUsedAt = &lastUsedStr
		}

		deviceInfos = append(deviceInfos, deviceInfo)
	}

	// 4. 构建响应
	rsp = &TOTPStatusRsp{
		Enabled:     activeDeviceCount > 0, // 有活跃设备则启用 2FA
		DeviceCount: activeDeviceCount,
		Devices:     deviceInfos,
	}

	log.Infoz("totp status retrieved successfully",
		zap.String("user_id", ctx.UserID),
		zap.Int("total_devices", len(devices)),
		zap.Int("active_devices", activeDeviceCount),
		zap.Bool("enabled", rsp.Enabled))

	return rsp, nil
}
