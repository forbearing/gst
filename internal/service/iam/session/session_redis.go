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

// InvalidateUserSessionsByUserID removes the Redis session mapping for a user and deletes the session value.
// It is best-effort: failures to talk to Redis do not block password updates.
func InvalidateUserSessionsByUserID(userID string) {
	if userID == "" {
		return
	}
	prefixedUserID := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, userID)
	sessionKey, err := redis.Cache[string]().Get(prefixedUserID)
	if err == nil && sessionKey != "" {
		_ = redis.Cache[modeliamsession.Session]().Delete(sessionKey)
	}
	_ = redis.Cache[string]().Delete(prefixedUserID)
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

// DeleteSessionBySessionID deletes the stored session and removes the user-to-session mapping
// only when it still points to the same session key.
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
		userKey := modeliamsession.SessionRedisKey(modeliamsession.SessionNamespace, session.UserID)
		mappedSessionKey, getErr := redis.Cache[string]().Get(userKey)
		switch {
		case getErr == nil && mappedSessionKey == sessionKey:
			if delErr := redis.Cache[string]().Delete(userKey); delErr != nil && !errors.Is(delErr, types.ErrEntryNotFound) {
				return session, delErr
			}
		case getErr == nil:
		case errors.Is(getErr, types.ErrEntryNotFound):
		default:
			return session, getErr
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
