package controller

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
	"unicode"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/authn/jwt"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/logger"
	"github.com/forbearing/gst/model"
	. "github.com/forbearing/gst/response"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/forbearing/gst/util"
	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
	cmap "github.com/orcaman/concurrent-map/v2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

var (
	loginRatelimiterMap  = cmap.New[*rate.Limiter]()
	signupRatelimiterMap = cmap.New[*rate.Limiter]()
)

type user struct{}

var User = new(user)

// Login 多次登陆之后，使用先前的 token 会报错 "access token not match"
func (*user) Login(c *gin.Context) {
	log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.Phase("Login"))
	limiter, found := loginRatelimiterMap.Get(c.ClientIP())
	if !found {
		limiter = rate.NewLimiter(rate.Every(1000*time.Millisecond), 10)
		loginRatelimiterMap.Set(c.ClientIP(), limiter)
	}
	if !limiter.Allow() {
		log.Error("too many login requests")
		JSON(c, NewCode(CodeTooManyRequests, http.StatusTooManyRequests, "too many login requests"))
		return
	}

	req := new(model.User)
	var err error
	if err = c.ShouldBindJSON(req); err != nil {
		log.Error(err)
		JSON(c, CodeInvalidLogin)
		return
	}
	users := make([]*model.User, 0)
	if err = database.Database[*model.User](types.NewDatabaseContext(c)).WithLimit(1).WithQuery(&model.User{Name: req.Name}).List(&users); err != nil {
		log.Error(err)
		JSON(c, CodeInvalidLogin)
		return
	}
	if len(users) == 0 {
		log.Error("not found any accounts")
		JSON(c, CodeInvalidLogin)
		return
	}
	if len(users) != 1 {
		log.Errorf("too many accounts: %d", len(users))
		JSON(c, CodeFailure)
		return
	}
	u := users[0]
	if err = comparePasswd(req.Password, u.Password); err != nil {
		log.Errorf("user password not match: %v", err)
		JSON(c, CodeInvalidLogin)
		return
	}
	// TODO: 把以前的 token 失效掉
	aToken, rToken, err := jwt.GenTokens(u.ID, req.Name, createSession(c))
	if err != nil {
		JSON(c, CodeFailure)
		return
	}
	u.Token = aToken
	u.AccessToken = aToken
	u.RefreshToken = rToken
	u.SessionID = util.UUID()
	fmt.Println("SessionId: ", u.SessionID)
	u.TokenExpiration = util.ValueOf(model.GormTime(time.Now().Add(config.App.AccessTokenExpireDuration)))
	writeLocalSessionAndCookie(c, aToken, rToken, u)
	// WARN: you must clean password before response to user.
	u.Password = ""

	u.LastLoginAt = util.ValueOf(model.GormTime(time.Now()))
	u.LastLoginIP = util.IPv6ToIPv4(c.ClientIP())
	if err = database.Database[*model.User](types.NewDatabaseContext(c)).UpdateByID(u.ID, "last_login", u.LastLoginAt); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	if err = database.Database[*model.User](types.NewDatabaseContext(c)).UpdateByID(u.ID, "last_login_ip", u.LastLoginIP); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	JSON(c, CodeSuccess, u)
}

func (*user) Logout(c *gin.Context) {
	log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.Phase("Logout"))
	_, claims, err := jwt.ParseTokenFromHeader(c.Request.Header)
	if err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	jwt.RevokeTokens(claims.Subject)

	JSON(c, CodeSuccess)
}

func (*user) RefreshToken(c *gin.Context) {
}

func (*user) Signup(c *gin.Context) {
	log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.Phase("Signup"))
	limiter, found := signupRatelimiterMap.Get(c.ClientIP())
	if !found {
		limiter = rate.NewLimiter(rate.Every(1000*time.Millisecond), 1)
		signupRatelimiterMap.Set(c.ClientIP(), limiter)
	}
	if !limiter.Allow() {
		log.Error("too many signup requests")
		JSON(c, NewCode(CodeTooManyRequests, http.StatusTooManyRequests, "too many signup requests"))
		return
	}

	req := new(model.User)
	var err error
	if err = c.ShouldBindJSON(req); err != nil {
		log.Error(err)
		JSON(c, CodeInvalidSignup)
		return
	}
	if err = validateUsername(req.Name); err != nil {
		log.Error(err)
		JSON(c, NewCode(CodeFailure, http.StatusBadRequest, err.Error()))
		return
	}
	if err = validatePassword(req.Password); err != nil {
		log.Error(err)
		JSON(c, NewCode(CodeFailure, http.StatusBadRequest, err.Error()))
		return
	}
	if req.Password != req.RePassword {
		log.Error("password and rePassword not the same")
		JSON(c, CodeInvalidSignup)
		return
	}

	users := make([]*model.User, 0)
	if err = database.Database[*model.User](types.NewDatabaseContext(c)).WithLimit(1).WithQuery(&model.User{Name: req.Name}).List(&users); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	if len(users) > 0 {
		JSON(c, CodeAlreadyExistsUser)
		return
	}
	hashedPasswd, err := encryptPasswd(req.Password)
	if err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	req.Password = hashedPasswd
	req.Status = 1
	req.ID = util.UUID()
	req.LastLoginAt = util.ValueOf(model.GormTime(time.Now()))
	req.LastLoginIP = util.IPv6ToIPv4(c.ClientIP())
	if err := database.Database[*model.User](types.NewDatabaseContext(c)).Create(req); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	JSON(c, CodeSuccess)
}

func (*user) ChangePasswd(c *gin.Context) {
	log := logger.Controller.WithControllerContext(types.NewControllerContext(c), consts.Phase("ChangePasswd"))

	req := new(model.User)
	if err := c.ShouldBindJSON(req); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	if len(req.ID) == 0 {
		log.Error(CodeNotFoundUserID)
		JSON(c, CodeNotFoundUserID)
		return
	}
	u := new(model.User)
	if err := database.Database[*model.User](types.NewDatabaseContext(c)).Get(u, req.ID); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	if len(u.ID) == 0 {
		log.Error(CodeNotFoundUser)
		JSON(c, CodeNotFoundUser)
		return
	}
	hashedPasswd, err := encryptPasswd(req.Password)
	if err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	if hashedPasswd != u.Password {
		log.Error(CodeOldPasswordNotMatch)
		JSON(c, CodeOldPasswordNotMatch)
		return
	}
	if req.NewPassword != req.RePassword {
		log.Error(CodeNewPasswordNotMatch)
		JSON(c, CodeNewPasswordNotMatch)
		return
	}
	hashedPasswd, err = encryptPasswd(req.NewPassword)
	if err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	u.Password = hashedPasswd
	if err = database.Database[*model.User](types.NewDatabaseContext(c)).Update(u); err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	_, claims, err := jwt.ParseTokenFromHeader(c.Request.Header)
	if err != nil {
		log.Error(err)
		JSON(c, CodeFailure)
		return
	}
	jwt.RevokeTokens(claims.Subject)
	JSON(c, CodeSuccess)
}

func encryptPasswd(pass string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pass), 8)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func comparePasswd(pass string, hashed string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pass))
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password too short")
	}
	var (
		hasNumber      = false
		hasLowerCase   = false
		hasUpperCase   = false
		hasSpecialChar = false
	)
	for _, c := range password {
		switch {
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsLower(c):
			hasLowerCase = true
		case unicode.IsUpper(c):
			hasUpperCase = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecialChar = true
		}
	}

	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasLowerCase {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasUpperCase {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasSpecialChar {
		return errors.New("password must contain at least one special character")
	}

	// if !hasNumber || !hasLowerCase || !hasUpperCase || !hasSpecialChar {
	// 	return fmt.Errorf("password too weak")
	// }
	return nil
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("username length must be between 3 and 32")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username) {
		return fmt.Errorf("username can only contain letters, numbers, underscores and hyphens")
	}
	return nil
}

func createSession(c *gin.Context) *model.Session {
	ua := useragent.New(c.Request.UserAgent())
	engineName, engineVersion := ua.Engine()
	browserName, browserVersion := ua.Browser()
	return &model.Session{
		UserID:         c.GetString(consts.CTX_USER_ID),
		Username:       c.GetString(consts.CTX_USERNAME),
		Platform:       ua.Platform(),
		OS:             ua.OS(),
		EngineName:     engineName,
		EngineVersion:  engineVersion,
		BrowserName:    browserName,
		BrowserVersion: browserVersion,
	}
}
