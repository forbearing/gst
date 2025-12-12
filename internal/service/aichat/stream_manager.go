package serviceaichat

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

// StreamManager manages ongoing streaming requests
type StreamManager struct {
	mu      sync.RWMutex
	streams map[string]context.CancelFunc // message_id -> cancel function
}

var globalStreamManager = &StreamManager{
	streams: make(map[string]context.CancelFunc),
}

// RegisterStream registers a streaming request with its cancel function
func (sm *StreamManager) RegisterStream(messageID string, cancel context.CancelFunc) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.streams[messageID]; exists {
		return errors.Newf("stream already exists for message: %s", messageID)
	}

	sm.streams[messageID] = cancel
	return nil
}

// CancelStream cancels a streaming request by message ID
func (sm *StreamManager) CancelStream(messageID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cancel, exists := sm.streams[messageID]
	if !exists {
		return errors.Newf("stream not found for message: %s", messageID)
	}

	if cancel != nil {
		cancel()
	}
	delete(sm.streams, messageID)
	return nil
}

// UnregisterStream removes a stream from the manager
func (sm *StreamManager) UnregisterStream(messageID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, messageID)
}

// GetStreamManager returns the global stream manager
func GetStreamManager() *StreamManager {
	return globalStreamManager
}
