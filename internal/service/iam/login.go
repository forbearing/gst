package serviceiam

import (
	"fmt"
	"net/http"
	"time"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	modeltwofa "github.com/forbearing/gst/internal/model/twofa"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
	"github.com/mssola/useragent"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	service.Base[*modeliam.Login, *modeliam.LoginReq, *modeliam.LoginRsp]
}

func (s *LoginService) Create(ctx *types.ServiceContext, req *modeliam.LoginReq) (rsp *modeliam.LoginRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	// return keycloakLogin(ctx, log, req)
	return localLogin(ctx, log, req)
}

func localLogin(ctx *types.ServiceContext, log types.Logger, req *modeliam.LoginReq) (rsp *modeliam.LoginRsp, err error) {
	// Validate input
	if req.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	var success bool
	ua := useragent.New(ctx.UserAgent)
	engineName, engineVersion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	defer func() {
		// write login log.
		if !success {
			if err = database.Database[*modellogmgmt.LoginLog](ctx.DatabaseContext()).Create(&modellogmgmt.LoginLog{
				Username: req.Username,
				ClientIP: ctx.ClientIP,
				Status:   modellogmgmt.LoginStatusFailure,
				Source:   ctx.Request.UserAgent(),
				Platform: fmt.Sprintf("%s %s", ua.Platform(), ua.OS()),
				Engine:   fmt.Sprintf("%s %s", engineName, engineVersion),
				Browser:  fmt.Sprintf("%s %s", browserName, browserVersion),
			}); err != nil {
				log.Warnz("failed to write login log", zap.Error(err))
			}
		}
	}()

	// Find user by username
	db := database.Database[*modeliam.User](ctx.DatabaseContext())
	users := make([]*modeliam.User, 0)
	if err = db.WithLimit(1).WithQuery(&modeliam.User{Username: req.Username}).List(&users); err != nil {
		log.Errorz("failed to query user", zap.Error(err))
		return nil, fmt.Errorf("invalid username or password")
	}
	if len(users) == 0 {
		log.Warnz("user not found", zap.String("username", req.Username))
		return nil, fmt.Errorf("invalid username or password")
	}
	user := users[0]

	// Check if user is enabled
	if user.Status == modeliam.UserStatusInactive {
		return nil, fmt.Errorf("user account is disabled")
	}
	if user.Status == modeliam.UserStatusLocked {
		return nil, fmt.Errorf("user account is locked")
	}

	// Verify password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		log.Warnz("invalid password", zap.String("username", req.Username))
		return nil, fmt.Errorf("invalid username or password")
	}

	// Check if user has 2FA enabled
	has2FA, err := checkUserHas2FA(ctx, user.ID)
	if err != nil {
		log.Errorz("failed to check 2FA status", zap.String("user_id", user.ID), zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	// If user has 2FA enabled, validate the 2FA code
	if has2FA {
		// Check if either TOTP code or backup code is provided
		if req.TOTPCode == "" && req.BackupCode == "" {
			log.Infoz("2FA required but no code provided", zap.String("username", req.Username))
			return nil, fmt.Errorf("2FA verification required")
		}

		// Validate TOTP code if provided
		if req.TOTPCode != "" {
			if err = validateTOTPCode(ctx, user.ID, req.TOTPCode); err != nil {
				log.Warnz("invalid TOTP code", zap.String("username", req.Username), zap.Error(err))
				return nil, fmt.Errorf("invalid 2FA code")
			}
			log.Infoz("TOTP code validated successfully", zap.String("username", req.Username))
		} else if req.BackupCode != "" {
			// Validate backup code if provided
			if err = validateBackupCode(ctx, user.ID, req.BackupCode); err != nil {
				log.Warnz("invalid backup code", zap.String("username", req.Username), zap.Error(err))
				return nil, fmt.Errorf("invalid backup code")
			}
			log.Infoz("backup code validated successfully", zap.String("username", req.Username))
		}
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	if err = db.Update(user); err != nil {
		log.Errorz("failed to update last login time", zap.Error(err))
		// Don't fail the login for this
	}

	// Query the group of the user
	group := new(modeliam.Group)
	_ = database.Database[*modeliam.Group](ctx.DatabaseContext()).Get(group, user.GroupID)

	// Parse user agent for session info

	// Create session
	sessionID := util.UUID()
	prefixedSessionID := modeliam.SessionRedisKey(modeliam.SessionNamespace, sessionID)
	prefixedUserID := modeliam.SessionRedisKey(modeliam.SessionNamespace, user.ID)

	// Create session data for local user
	sessionData := modeliam.Session{
		UserID:      user.ID,
		Username:    user.Username,
		Email:       util.Deref(user.Email),
		OS:          ua.OS(),
		Platform:    ua.Platform(),
		EngineName:  engineName,
		BrowserName: browserName,
		UserInfo: map[string]any{
			"user_id":    user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"group":      group,
		},
	}

	expire := 8 * time.Hour
	// Store session in Redis
	if err = redis.Cache[modeliam.Session]().Set(prefixedSessionID, sessionData, expire); err != nil {
		log.Errorz("failed to set session in redis", zap.Error(err))
		return nil, fmt.Errorf("failed to set session in redis")
	}
	// Store session id in Redis
	if err = redis.Cache[string]().Set(prefixedUserID, prefixedSessionID, expire); err != nil {
		log.Errorz("failed to set session id in redis", zap.Error(err))
		return nil, fmt.Errorf("failed to set session id in redis")
	}

	// Set cookie
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   8 * 60 * 60,          // 8 hours
		HttpOnly: true,                 // More secure
		Secure:   false,                // Set to false for local development
		SameSite: http.SameSiteLaxMode, // Lax mode
	})

	log.Infoz("user logged in successfully", zap.String("username", req.Username), zap.String("user_id", user.ID))

	// write login log
	success = true
	if err = database.Database[*modellogmgmt.LoginLog](ctx.DatabaseContext()).Create(&modellogmgmt.LoginLog{
		UserID:   user.ID,
		Username: user.Username,
		ClientIP: ctx.ClientIP,
		Status:   modellogmgmt.LoginStatusSuccess,

		Source:   ctx.Request.UserAgent(),
		Platform: fmt.Sprintf("%s %s", ua.Platform(), ua.OS()),
		Engine:   fmt.Sprintf("%s %s", engineName, engineVersion),
		Browser:  fmt.Sprintf("%s %s", browserName, browserVersion),
	}); err != nil {
		log.Warnz("failed to write login log", zap.Error(err))
	}

	return &modeliam.LoginRsp{
		SessionID: sessionID,
	}, nil
}

// checkUserHas2FA checks if the user has active TOTP devices
func checkUserHas2FA(ctx *types.ServiceContext, userID string) (bool, error) {
	db := database.Database[*modeltwofa.TOTPDevice](ctx.DatabaseContext())
	devices := make([]*modeltwofa.TOTPDevice, 0)

	if err := db.WithQuery(&modeltwofa.TOTPDevice{
		UserID:   userID,
		IsActive: true,
	}).List(&devices); err != nil {
		return false, fmt.Errorf("failed to query TOTP devices: %w", err)
	}

	return len(devices) > 0, nil
}

// validateTOTPCode validates the provided TOTP code for the user
func validateTOTPCode(ctx *types.ServiceContext, userID, code string) error {
	if code == "" {
		return fmt.Errorf("TOTP code is required")
	}

	db := database.Database[*modeltwofa.TOTPDevice](ctx.DatabaseContext())
	devices := make([]*modeltwofa.TOTPDevice, 0)

	if err := db.WithQuery(&modeltwofa.TOTPDevice{
		UserID:   userID,
		IsActive: true,
	}).List(&devices); err != nil {
		return fmt.Errorf("failed to query TOTP devices: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no active TOTP devices found")
	}

	// Try to validate the code against all active devices
	for _, device := range devices {
		if totp.Validate(code, device.Secret) {
			return nil
		}
	}

	return fmt.Errorf("invalid TOTP code")
}

// validateBackupCode validates the provided backup code for the user
func validateBackupCode(ctx *types.ServiceContext, userID, code string) error {
	if code == "" {
		return fmt.Errorf("backup code is required")
	}

	db := database.Database[*modeltwofa.TOTPDevice](ctx.DatabaseContext())
	devices := make([]*modeltwofa.TOTPDevice, 0)

	if err := db.WithQuery(&modeltwofa.TOTPDevice{
		UserID:   userID,
		IsActive: true,
	}).List(&devices); err != nil {
		return fmt.Errorf("failed to query TOTP devices: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no active TOTP devices found")
	}

	// Check backup codes for all active devices
	for _, device := range devices {
		if len(device.BackupCodes) > 0 {
			for i, backupCode := range device.BackupCodes {
				if backupCode == code {
					// Mark backup code as used by removing it
					updatedCodes := make([]string, 0, len(device.BackupCodes)-1)
					for j, bc := range device.BackupCodes {
						if j != i {
							updatedCodes = append(updatedCodes, bc)
						}
					}
					device.BackupCodes = updatedCodes

					// Update the device in database
					if err := db.Update(device); err != nil {
						return fmt.Errorf("failed to update backup codes: %w", err)
					}

					return nil
				}
			}
		}
	}

	return fmt.Errorf("invalid backup code")
}

// func keycloakLogin(ctx *types.ServiceContext, log types.Logger, req *iam.LoginReq) (rsp *iam.LoginRsp, err error) {
// 	kccfg := config.Get[configx.Keycloak]()
//
// 	// keycloak 校验用户名和密码
// 	tokens, err := keycloak.IdentityLogin(log, req.Username, req.Password)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// 获取用户信息
// 	userInfo, err := keycloak.UserInfo(log, tokens.AccessToken)
// 	if err != nil {
// 		log.Error(err)
// 		return nil, err
// 	}
//
// 	// 解析 token、解析前端浏览器信息
// 	jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", kccfg.Addr, kccfg.Realm)
// 	claims := &helper.Claims{}
// 	if _, err := jwt.ParseWithClaims(tokens.AccessToken, claims, helper.KeyFuncForKeycloak(jwksURL)); err != nil {
// 		log.Error(err)
// 		return nil, err
// 	}
// 	ua := useragent.New(ctx.UserAgent)
// 	engineName, _ := ua.Engine()
// 	browserName, _ := ua.Browser()
//
// 	// 存入 redis
// 	sessionID := util.UUID()
// 	redisKey := helper.SessionRedisKey(iam.SessionNamespace, sessionID)
// 	if err := redis.Cache[iam.Session]().Set(redisKey, iam.Session{
// 		UserID:      claims.Sub,
// 		Username:    claims.PreferredUsername,
// 		Email:       claims.Email,
// 		OS:          ua.OS(),
// 		Platform:    ua.Platform(),
// 		EngineName:  engineName,
// 		BrowserName: browserName,
// 		Token:       *tokens,
// 		UserInfo:    userInfo,
// 	}, 8*time.Hour); err != nil {
// 		log.Error("failed to set session in redis", zap.Error(err))
// 		return nil, fmt.Errorf("failed to set session in redis")
// 	}
//
// 	// 设置前端 cookie
// 	http.SetCookie(ctx.Writer, &http.Cookie{
// 		Name:  "session_id",
// 		Value: sessionID,
// 		Path:  "/",
// 		// MaxAge:   tokens.RefreshExpiresIn,
// 		MaxAge:   8 * 60 * 60,          // 8 hours
// 		HttpOnly: true,                 // 建议设为 true，更安全
// 		Secure:   false,                // 本地开发设为 false
// 		SameSite: http.SameSiteLaxMode, // 改为 Lax
// 	})
//
// 	// 返回给前端
// 	rsp = &iam.LoginRsp{
// 		SessionID: sessionID,
// 	}
//
// 	return rsp, nil
// }
