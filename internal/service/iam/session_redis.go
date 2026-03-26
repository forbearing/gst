package serviceiam

import (
	"github.com/cockroachdb/errors"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/types"
)

// invalidateUserSessionsByUserID removes the Redis session mapping for a user and deletes the session value.
// It is best-effort: failures to talk to Redis do not block password updates.
func invalidateUserSessionsByUserID(userID string) {
	if userID == "" {
		return
	}
	prefixedUserID := modeliam.SessionRedisKey(modeliam.SessionNamespace, userID)
	sessionKey, err := redis.Cache[string]().Get(prefixedUserID)
	if err == nil && sessionKey != "" {
		_ = redis.Cache[modeliam.Session]().Delete(sessionKey)
	}
	_ = redis.Cache[string]().Delete(prefixedUserID)
}

// syncSessionMustChangePassword updates the stored session after the user clears MustChangePassword in the database.
func syncSessionMustChangePassword(sessionID string, mustChange bool) error {
	if sessionID == "" {
		return nil
	}
	sessionKey := modeliam.SessionRedisKey(modeliam.SessionNamespace, sessionID)
	session, err := redis.Cache[modeliam.Session]().Get(sessionKey)
	if err != nil {
		if errors.Is(err, types.ErrEntryNotFound) {
			return nil
		}
		return err
	}
	if session.UserInfo == nil {
		session.UserInfo = map[string]any{}
	}
	session.UserInfo["must_change_password"] = mustChange
	return redis.Cache[modeliam.Session]().Set(sessionKey, session, getSessionExpiration())
}
