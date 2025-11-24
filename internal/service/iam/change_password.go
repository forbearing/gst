package serviceiam

import (
	"fmt"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"golang.org/x/crypto/bcrypt"
)

type ChangePasswordService struct {
	service.Base[*modeliam.ChangePassword, *modeliam.ChangePasswordReq, *modeliam.ChangePasswordRsp]
}

func (s *ChangePasswordService) Create(ctx *types.ServiceContext, req *modeliam.ChangePasswordReq) (rsp *modeliam.ChangePasswordRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())
	log.Info("changepassword create")

	// Get session_id from cookie
	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error("failed to get session_id from cookie", err)
		return nil, fmt.Errorf("authentication required")
	}

	// Get session from Redis
	redisKey := SessionRedisKey(modeliam.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliam.Session]().Get(redisKey)
	if err != nil {
		log.Error("failed to get session from redis", err)
		return nil, fmt.Errorf("invalid session")
	}

	// Get user from database
	db := database.Database[*modeliam.User](ctx.DatabaseContext())
	users := make([]*modeliam.User, 0)
	if err = db.WithLimit(1).WithQuery(&modeliam.User{Username: session.Username}).List(&users); err != nil {
		log.Error("failed to query user", err)
		return nil, fmt.Errorf("database error")
	}
	if len(users) == 0 {
		log.Error("user not found", "username", session.Username)
		return nil, fmt.Errorf("user not found")
	}
	user := users[0]

	// Verify old password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		log.Error("old password verification failed", "username", user.Username)
		return nil, fmt.Errorf("old password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash new password", err)
		return nil, fmt.Errorf("failed to process new password")
	}

	// Update password in database
	user.PasswordHash = string(hashedPassword)
	if err := db.Update(user); err != nil {
		log.Error("failed to update password", err)
		return nil, fmt.Errorf("failed to update password")
	}

	log.Info("password changed successfully", "username", user.Username)
	return &modeliam.ChangePasswordRsp{Msg: "password changed successfully"}, nil
}
