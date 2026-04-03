package serviceiamsession

import (
	"net/http"
	"sort"

	"github.com/cockroachdb/errors"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// SessionsListService handles retrieval of all active sessions for the current authenticated user.
type SessionsListService struct {
	service.Base[*model.Empty, *modeliamsession.SessionsListReq, *modeliamsession.SessionsListRsp]
}

// SessionsDeleteService handles invalidation of a specified session for the current authenticated user.
type SessionsDeleteService struct {
	service.Base[*model.Empty, *modeliamsession.SessionsDeleteReq, *modeliamsession.SessionsDeleteRsp]
}

// List returns all active sessions for the current authenticated user.
func (s *SessionsListService) List(ctx *types.ServiceContext, req *modeliamsession.SessionsListReq) (rsp *modeliamsession.SessionsListRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	// GetCurrentSession already guarantees that the resolved session is bound to
	// an authenticated user, so the service can directly use currentSession.UserID.
	sessionID, currentSession, err := GetCurrentSession(ctx)
	if err != nil {
		log.Error("failed to get current session", err)
		return nil, err
	}

	sessionIDs, err := listUserSessionIDs(currentSession.UserID)
	if err != nil {
		log.Error("failed to list user sessions", err)
		return nil, err
	}

	items := make([]modeliamsession.CurrentSession, 0, len(sessionIDs))
	for i := range sessionIDs {
		sessionKey := modeliamsession.SessionIDKey(sessionIDs[i])
		session, getErr := redis.Cache[modeliamsession.Session]().Get(sessionKey)
		if getErr != nil {
			if errors.Is(getErr, types.ErrEntryNotFound) {
				_ = redis.ZRem(modeliamsession.SessionUserKey(currentSession.UserID), sessionIDs[i])
				continue
			}
			log.Error("failed to load session from redis", getErr)
			return nil, getErr
		}
		items = append(items, buildCurrentSession(session, sessionID))
	}

	sort.Slice(items, func(i, j int) bool {
		left := items[i].LastSeenAt
		if left.IsZero() {
			left = items[i].IssuedAt
		}
		right := items[j].LastSeenAt
		if right.IsZero() {
			right = items[j].IssuedAt
		}
		if left.Equal(right) {
			return items[i].ID > items[j].ID
		}
		return left.After(right)
	})

	return &modeliamsession.SessionsListRsp{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// Delete invalidates a specified session for the current authenticated user.
// When the route id is "others", it keeps the current session active and
// revokes every other indexed session of the same user. The endpoint remains
// idempotent: deleting a missing session still returns success.
func (s *SessionsDeleteService) Delete(ctx *types.ServiceContext, req *modeliamsession.SessionsDeleteReq) (rsp *modeliamsession.SessionsDeleteRsp, err error) {
	log := s.WithServiceContext(ctx, ctx.GetPhase())

	currentSessionID, currentSession, err := GetCurrentSession(ctx)
	if err != nil {
		log.Error("failed to get current session", err)
		return nil, err
	}

	targetSessionID := ctx.Params["id"]
	if targetSessionID == "" {
		return nil, types.NewServiceError(http.StatusBadRequest, "session id is required")
	}
	if targetSessionID == "others" {
		// DELETE /api/iam/sessions/others is a bulk self-service logout for
		// secondary sessions. The current cookie-backed session must survive so
		// the caller can continue using the API after the request completes.
		if err = DeleteOtherSessions(currentSession.UserID, currentSessionID); err != nil {
			log.Error("failed to delete other sessions", err)
			return nil, err
		}
		return &modeliamsession.SessionsDeleteRsp{}, nil
	}

	targetSession, err := redis.Cache[modeliamsession.Session]().Get(modeliamsession.SessionIDKey(targetSessionID))
	if err != nil {
		if errors.Is(err, types.ErrEntryNotFound) {
			if targetSessionID == currentSessionID {
				ctx.SetCookie("session_id", "", -1, "/", "", false, true)
			}
			return &modeliamsession.SessionsDeleteRsp{}, nil
		}
		log.Error("failed to load target session", err)
		return nil, err
	}
	if targetSession.UserID != currentSession.UserID {
		return nil, types.NewServiceError(http.StatusForbidden, "forbidden")
	}

	if _, err = DeleteSession(targetSessionID); err != nil {
		if errors.Is(err, types.ErrEntryNotFound) {
			if targetSessionID == currentSessionID {
				ctx.SetCookie("session_id", "", -1, "/", "", false, true)
			}
			return &modeliamsession.SessionsDeleteRsp{}, nil
		}
		log.Error("failed to delete target session", err)
		return nil, err
	}
	if targetSessionID == currentSessionID {
		ctx.SetCookie("session_id", "", -1, "/", "", false, true)
	}

	return &modeliamsession.SessionsDeleteRsp{}, nil
}
