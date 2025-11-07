package twofa

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"

	. "github.com/forbearing/gst/dsl"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
)

var _ types.Module[*TOTPBind, *TOTPBind, *TOTPBindRsp] = (*TOTPBindModule)(nil)

type TOTPBindModule struct{}

func (*TOTPBindModule) Service() types.Service[*TOTPBind, *TOTPBind, *TOTPBindRsp] {
	return &TOTPBindService{}
}
func (*TOTPBindModule) Route() string { return "2fa/totp/bind" }
func (*TOTPBindModule) Pub() bool     { return false }
func (*TOTPBindModule) Param() string { return "id" }

// TOTPBind 绑定 TOTP 设备
type TOTPBind struct {
	model.Empty
}

func (TOTPBind) Design() {
	Route("2fa/totp/bind", func() {
		Create(func() {
			Enabled(true)
			Service(true)
			Result[*TOTPBindRsp]()
		})
	})
}

type TOTPBindRsp struct {
	Secret      string `json:"secret"`        // Base32 编码的 TOTP 密钥
	OtpauthURL  string `json:"otpauth_url"`   // TOTP 认证 URL
	QRCodeImage string `json:"qr_code_image"` // Base64 编码的二维码图片数据
	Issuer      string `json:"issuer"`        // 应用发行者名称（如 "Nebula"）
	AccountName string `json:"account_name"`  // 用户账户名称
}

type TOTPBindService struct {
	service.Base[*TOTPBind, *TOTPBind, *TOTPBindRsp]
}

func (t *TOTPBindService) Create(ctx *types.ServiceContext, req *TOTPBind) (rsp *TOTPBindRsp, err error) {
	log := t.WithServiceContext(ctx, ctx.GetPhase())

	// 获取当前用户信息
	if len(ctx.UserID) == 0 {
		log.Errorz("user_id not found in context")
		return nil, types.NewServiceError(http.StatusUnauthorized, "authentication required")
	}

	if len(ctx.Username) == 0 {
		log.Errorz("username not found in context")
		return nil, types.NewServiceError(http.StatusUnauthorized, "authentication required")
	}

	log.Infoz("generating TOTP for user", zap.String("user_id", ctx.UserID), zap.String("username", ctx.Username))

	// 生成 TOTP 密钥
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Nebula",
		AccountName: ctx.Username,
		SecretSize:  32, // 32 bytes = 256 bits
	})
	if err != nil {
		log.Errorz("failed to generate TOTP key", zap.Error(err))
		return nil, fmt.Errorf("failed to generate TOTP key")
	}

	// 生成 QR 码 URL
	qrCodeURL := key.URL()
	log.Infoz("generated QR code URL", zap.String("url", qrCodeURL))

	// 生成 QR 码图片数据
	qrCodeImage, err := generateQRCode(qrCodeURL)
	if err != nil {
		log.Errorz("failed to generate QR code image", zap.Error(err))
		return nil, fmt.Errorf("failed to generate QR code image")
	}

	rsp = &TOTPBindRsp{
		Secret:      key.Secret(),
		OtpauthURL:  qrCodeURL,
		QRCodeImage: qrCodeImage,
		Issuer:      "Nebula",
		AccountName: ctx.Username,
	}

	log.Infoz("TOTP bind response generated successfully",
		zap.String("user_id", ctx.UserID))

	return rsp, nil
}

// generateQRCode 生成 QR 码的 Data URL
func generateQRCode(url string) (string, error) {
	// 生成 QR 码 PNG 数据
	qrBytes, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}

	// 转换为 base64 Data URL
	var buf bytes.Buffer
	buf.WriteString("data:image/png;base64,")
	buf.WriteString(base64.StdEncoding.EncodeToString(qrBytes))

	return buf.String(), nil
}
