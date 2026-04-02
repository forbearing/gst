package modeliamsession

import (
	"fmt"
	"strings"
	"time"
)

// SessionNamespacePrefix is the shared Redis key prefix for IAM session storage.
const SessionNamespacePrefix = "iam:session"

// SessionIDNamespace stores session snapshots by session ID.
const SessionIDNamespace = SessionNamespacePrefix + ":id"

// SessionUserNamespace stores the session index set by user ID.
const SessionUserNamespace = SessionNamespacePrefix + ":user"

type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "active"
	SessionStatusRevoked SessionStatus = "revoked"
	SessionStatusExpired SessionStatus = "expired"
)

// Session stores the authenticated session snapshot used by IAM middleware and session APIs.
type Session struct {
	ID string `json:"id"`

	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   string `json:"status"`

	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	GroupID   string  `json:"group_id,omitempty"`
	GroupName string  `json:"group_name,omitempty"`

	MustChangePassword bool `json:"must_change_password"`

	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`

	Platform    string `json:"platform"`
	OS          string `json:"os"`
	EngineName  string `json:"engine_name"`
	BrowserName string `json:"browser_name"`

	State      SessionStatus `json:"state"`
	IssuedAt   time.Time     `json:"issued_at"`
	LastSeenAt time.Time     `json:"last_seen_at"`
	ExpiresAt  time.Time     `json:"expires_at"`

	Token Token `json:"token"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	ExpiresIn        int `json:"expires_in"`
	RefreshExpiresIn int `json:"refresh_expires_in"`

	TokenType string `json:"token_type"`
	Scope     string `json:"scope"`

	NotBeforePolicy int    `json:"not-before-policy"`
	SessionState    string `json:"session_state"`
}

// SessionRedisKey builds a Redis key for the specified namespace and identifier.
func SessionRedisKey(namespace, id string) string {
	return fmt.Sprintf("%s:%s", namespace, id)
}

// SessionUserRedisKey builds the Redis key for the indexed session set of a user.
func SessionUserRedisKey(userID string) string {
	return SessionRedisKey(SessionUserNamespace, userID)
}

// SessionID extracts the identifier from a namespaced Redis key.
func SessionID(redisKey string, namespace string) string {
	return strings.TrimPrefix(redisKey, fmt.Sprintf("%s:", namespace))
}
