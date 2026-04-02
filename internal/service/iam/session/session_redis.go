package serviceiamsession

import (
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/types"
)

var (
	sessionExpiration   time.Duration
	sessionExpirationMu sync.RWMutex
)

// listUserSessionIDsByUserID loads all indexed session ids for a user.
func listUserSessionIDsByUserID(userID string) ([]string, error) {
	if userID == "" {
		return make([]string, 0), nil
	}
	userKey := modeliamsession.SessionUserRedisKey(userID)
	return redis.ZRange(userKey, 0, -1)
}

// TrackUserSession adds the session id into the user's indexed session set.
func TrackUserSession(session modeliamsession.Session) error {
	if session.UserID == "" || session.ID == "" {
		return nil
	}
	userKey := modeliamsession.SessionUserRedisKey(session.UserID)
	score := float64(session.IssuedAt.UnixMilli())
	if err := redis.ZAdd(userKey, score, session.ID); err != nil {
		return err
	}
	return redis.Expire(userKey, GetSessionExpiration())
}

// InvalidateUserSessionsByUserID removes all indexed sessions for a user.
// It is best-effort: failures to talk to Redis do not block password updates.
func InvalidateUserSessionsByUserID(userID string) {
	if userID == "" {
		return
	}
	sessionIDs, err := listUserSessionIDsByUserID(userID)
	if err == nil {
		for i := range sessionIDs {
			sessionKey := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionIDs[i])
			_ = redis.Cache[modeliamsession.Session]().Delete(sessionKey)
		}
	}
	_ = redis.Del(modeliamsession.SessionUserRedisKey(userID))
}

// SyncSessionMustChangePassword updates the stored session after the user clears MustChangePassword in the database.
func SyncSessionMustChangePassword(sessionID string, mustChange bool) error {
	if sessionID == "" {
		return nil
	}
	sessionKey := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	if err != nil {
		if errors.Is(err, types.ErrEntryNotFound) {
			return nil
		}
		return err
	}
	session.MustChangePassword = mustChange
	return redis.Cache[modeliamsession.Session]().Set(sessionKey, session, GetSessionExpiration())
}

// DeleteSessionBySessionID deletes the stored session and removes the indexed user-session relation.
func DeleteSessionBySessionID(sessionID string) (modeliamsession.Session, error) {
	if sessionID == "" {
		return modeliamsession.Session{}, nil
	}

	sessionKey := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	if err != nil {
		return modeliamsession.Session{}, err
	}
	if err = redis.Cache[modeliamsession.Session]().Delete(sessionKey); err != nil && !errors.Is(err, types.ErrEntryNotFound) {
		return session, err
	}

	if session.UserID != "" {
		userKey := modeliamsession.SessionUserRedisKey(session.UserID)
		if err = redis.ZRem(userKey, sessionID); err != nil {
			return session, err
		}
	}

	return session, nil
}

// GetSessionExpiration returns the configured session expiration time.
// If not configured, it returns the default value of 8 hours.
func GetSessionExpiration() time.Duration {
	sessionExpirationMu.RLock()
	defer sessionExpirationMu.RUnlock()
	if sessionExpiration == 0 {
		return 8 * time.Hour
	}
	return sessionExpiration
}

// SetSessionExpiration sets the session expiration time for iam module.
// This function should be called during module registration.
func SetSessionExpiration(expiration time.Duration) {
	sessionExpirationMu.Lock()
	defer sessionExpirationMu.Unlock()
	sessionExpiration = expiration
}
