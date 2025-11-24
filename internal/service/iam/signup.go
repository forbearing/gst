package serviceiam

import (
	"fmt"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type SignupService struct {
	service.Base[*modeliam.Signup, *modeliam.SignupReq, *modeliam.SignupRsp]
}

func (s *SignupService) Create(ctx *types.ServiceContext, req *modeliam.SignupReq) (rsp *modeliam.SignupRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	// Validate input
	if req.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if req.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if req.Password != req.RePassword {
		return nil, fmt.Errorf("passwords do not match")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters long")
	}

	// Check if username already exists
	db := database.Database[*modeliam.User](ctx.DatabaseContext())
	existingUsers := make([]*modeliam.User, 0)
	if err := db.WithLimit(1).WithQuery(&modeliam.User{Username: req.Username}).List(&existingUsers); err != nil {
		log.Error("failed to check existing user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user")
	}
	if len(existingUsers) > 0 {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("failed to create user")
	}

	// Create new user
	newUser := &modeliam.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
	}

	// Set optional fields
	if req.Email != "" {
		newUser.Email = &req.Email
	}
	if req.FirstName != "" {
		newUser.FirstName = &req.FirstName
	}
	if req.LastName != "" {
		newUser.LastName = &req.LastName
	}

	// Save to database
	if err := db.Create(newUser); err != nil {
		log.Error("failed to create user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user")
	}

	log.Info("user created successfully", zap.String("username", req.Username), zap.String("user_id", newUser.ID))

	return &modeliam.SignupRsp{
		UserID:   newUser.ID,
		Username: newUser.Username,
		Message:  "User created successfully",
	}, nil
}
