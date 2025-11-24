package servicetwofa

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"

	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
)

type TOTPBindService struct {
	service.Base[*modeltwofa.TOTPBind, *modeltwofa.TOTPBind, *modeltwofa.TOTPBindRsp]
}

func (t *TOTPBindService) Create(ctx *types.ServiceContext, req *modeltwofa.TOTPBind) (rsp *modeltwofa.TOTPBindRsp, err error) {
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

	rsp = &modeltwofa.TOTPBindRsp{
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
