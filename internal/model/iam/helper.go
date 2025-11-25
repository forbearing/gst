package modeliam

import (
	"fmt"
	"strings"
)

// SessionRedisKey 构造一个 redis key
func SessionRedisKey(namespace, id string) string {
	return fmt.Sprintf("%s:%s", namespace, id)
}

// SessionID 从 redis key 中获取 session id
func SessionID(redisKey string, namespace string) string {
	return strings.TrimPrefix(redisKey, fmt.Sprintf("%s:", namespace))
}
