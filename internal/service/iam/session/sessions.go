package serviceiamsession

import (
	"sort"

	"github.com/cockroachdb/errors"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types"
)

// SessionsService handles retrieval of all active sessions for the current authenticated user.
type SessionsService struct {
	service.Base[*model.Empty, *modeliamsession.SessionsReq, *modeliamsession.SessionsRsp]
}

// List returns all active sessions for the current authenticated user.
func (s *SessionsService) List(ctx *types.ServiceContext, req *modeliamsession.SessionsReq) (rsp *modeliamsession.SessionsRsp, err error) {
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

	return &modeliamsession.SessionsRsp{
		Items: items,
		Total: int64(len(items)),
	}, nil
}
