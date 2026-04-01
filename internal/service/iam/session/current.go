package serviceiamsession

import (
	"net/http"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

// CurrentService handles retrieval and invalidation of the current authenticated session.
type CurrentService struct {
	service.Base[*modeliamsession.Current, *modeliamsession.CurrentReq, *modeliamsession.CurrentRsp]
}

// List returns the current authenticated session together with the latest user snapshot.
func (s *CurrentService) List(ctx *types.ServiceContext, req *modeliamsession.CurrentReq) (rsp *modeliamsession.CurrentRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error(err)
		return nil, types.NewServiceError(http.StatusUnauthorized, err.Error())
	}

	session, e := redis.Cache[modeliamsession.Session]().Get(modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionID))
	if e != nil {
		log.Error("session not exists")
		return nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "session not exists", e)
	}

	user := new(modeliam.User)
	if err := database.Database[*modeliam.User](ctx.DatabaseContext()).Get(user, session.UserID); err != nil || user.GetID() == "" {
		log.Error("failed to load user for current session")
		return nil, types.NewServiceError(http.StatusUnauthorized, "session invalid")
	}
	switch user.Status {
	case modeliam.UserStatusInactive:
		return nil, types.NewServiceError(http.StatusForbidden, "", response.CodeAccountInactive)
	case modeliam.UserStatusLocked:
		return nil, types.NewServiceError(http.StatusForbidden, "", response.CodeAccountLocked)
	}

	groupName := session.GroupName
	if session.GroupID != "" && groupName == "" {
		group := new(modeliam.Group)
		if err := database.Database[*modeliam.Group](ctx.DatabaseContext()).Get(group, session.GroupID); err == nil {
			groupName = group.Name
		}
	}

	return buildCurrentRsp(session, sessionID, &modeliamsession.CurrentPrincipal{
		UserID:             user.ID,
		Username:           user.Username,
		Email:              util.Deref(user.Email),
		FirstName:          user.FirstName,
		LastName:           user.LastName,
		GroupID:            user.GroupID,
		GroupName:          groupName,
		Status:             string(user.Status),
		MustChangePassword: user.MustChangePassword,
	}), nil
}

// Delete invalidates the current authenticated session and clears the session cookie.
func (s *CurrentService) Delete(ctx *types.ServiceContext, req *modeliamsession.CurrentReq) (rsp *modeliamsession.CurrentRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error(err)
		return nil, types.NewServiceError(http.StatusUnauthorized, err.Error())
	}

	session, err := DeleteSessionBySessionID(sessionID)
	if err != nil {
		log.Error("failed to delete current session", err)
		return nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "session not exists", err)
	}

	ctx.SetCookie("session_id", "", -1, "/", "", false, true)

	return buildCurrentRsp(session, sessionID, &modeliamsession.CurrentPrincipal{
		UserID:             session.UserID,
		Username:           session.Username,
		Email:              session.Email,
		FirstName:          session.FirstName,
		LastName:           session.LastName,
		GroupID:            session.GroupID,
		GroupName:          session.GroupName,
		Status:             session.Status,
		MustChangePassword: session.MustChangePassword,
	}), nil
}

// buildCurrentRsp builds the API response for current session endpoints from the stored session snapshot.
func buildCurrentRsp(session modeliamsession.Session, fallbackSessionID string, principal *modeliamsession.CurrentPrincipal) *modeliamsession.CurrentRsp {
	currentSessionID := session.ID
	if currentSessionID == "" {
		currentSessionID = fallbackSessionID
	}
	state := session.State
	if state == "" {
		state = modeliamsession.SessionStatusActive
	}

	if principal == nil {
		principal = &modeliamsession.CurrentPrincipal{}
	}

	return &modeliamsession.CurrentRsp{
		Session: modeliamsession.CurrentSession{
			ID:          currentSessionID,
			State:       state,
			IssuedAt:    session.IssuedAt,
			LastSeenAt:  session.LastSeenAt,
			ExpiresAt:   session.ExpiresAt,
			ClientIP:    session.ClientIP,
			UserAgent:   session.UserAgent,
			Platform:    session.Platform,
			OS:          session.OS,
			EngineName:  session.EngineName,
			BrowserName: session.BrowserName,
			IsCurrent:   true,
		},
		Principal: *principal,
	}
}
