package serviceiamsession

import (
	"net/http"

	"github.com/forbearing/gst/database"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/util"
)

// CurrentListService handles retrieval of the current authenticated session.
type CurrentListService struct {
	service.Base[*model.Empty, *modeliamsession.CurrentListReq, *modeliamsession.CurrentListRsp]
}

// List returns the current authenticated session together with the latest user snapshot.
func (s *CurrentListService) List(ctx *types.ServiceContext, req *modeliamsession.CurrentListReq) (rsp *modeliamsession.CurrentListRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, session, err := GetCurrentSession(ctx)
	if err != nil {
		log.Error("failed to get current session", err)
		return nil, err
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

	return buildCurrentListRsp(session, sessionID, &modeliamsession.CurrentPrincipal{
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

// CurrentDeleteService handles invalidation of the current authenticated session.
type CurrentDeleteService struct {
	service.Base[*model.Empty, *modeliamsession.CurrentDeleteReq, *modeliamsession.CurrentDeleteRsp]
}

// Delete invalidates the current authenticated session and clears the session cookie.
func (s *CurrentDeleteService) Delete(ctx *types.ServiceContext, req *modeliamsession.CurrentDeleteReq) (rsp *modeliamsession.CurrentDeleteRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	sessionID, err := ctx.Cookie("session_id")
	if err != nil {
		log.Error(err)
		return nil, types.NewServiceError(http.StatusUnauthorized, err.Error())
	}

	if _, err = DeleteSession(sessionID); err != nil {
		log.Error("failed to delete current session", err)
		return nil, types.NewServiceErrorWithCause(http.StatusUnauthorized, "session not exists", err)
	}

	ctx.SetCookie("session_id", "", -1, "/", "", false, true)

	return &modeliamsession.CurrentDeleteRsp{}, nil
}

// buildCurrentListRsp builds the API response for getting the current session from the stored session snapshot.
func buildCurrentListRsp(session modeliamsession.Session, fallbackSessionID string, principal *modeliamsession.CurrentPrincipal) *modeliamsession.CurrentListRsp {
	if principal == nil {
		principal = &modeliamsession.CurrentPrincipal{}
	}

	return &modeliamsession.CurrentListRsp{
		Session:   buildCurrentSession(session, fallbackSessionID),
		Principal: *principal,
	}
}

// buildCurrentSession builds the response snapshot for a session summary endpoint.
func buildCurrentSession(session modeliamsession.Session, currentSessionID string) modeliamsession.CurrentSession {
	sessionID := session.ID
	if sessionID == "" {
		sessionID = currentSessionID
	}
	state := session.State
	if state == "" {
		state = modeliamsession.SessionStatusActive
	}

	return modeliamsession.CurrentSession{
		ID:          sessionID,
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
		IsCurrent:   sessionID == currentSessionID,
	}
}
