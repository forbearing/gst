package serviceaichat

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetStreamManager(t *testing.T) {
	t.Run("returns singleton instance", func(t *testing.T) {
		manager1 := GetStreamManager()
		manager2 := GetStreamManager()
		require.NotNil(t, manager1)
		require.NotNil(t, manager2)
		require.Equal(t, manager1, manager2)
	})
}

func TestRegisterStream(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("register new stream", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-1", cancel)
		require.NoError(t, err)

		sm.mu.RLock()
		_, exists := sm.streams["msg-1"]
		sm.mu.RUnlock()
		require.True(t, exists)
	})

	t.Run("register duplicate stream returns error", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		// First registration should succeed
		err := sm.RegisterStream("msg-2", cancel)
		require.NoError(t, err)

		// Second registration should fail
		_, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		err = sm.RegisterStream("msg-2", cancel2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "stream already exists")
	})

	t.Run("register multiple streams", func(t *testing.T) {
		for i := range 10 {
			_, cancel := context.WithCancel(context.Background())
			defer cancel()
			err := sm.RegisterStream("msg-multi-"+string(rune(i)), cancel)
			require.NoError(t, err)
		}

		sm.mu.RLock()
		count := len(sm.streams)
		sm.mu.RUnlock()
		require.GreaterOrEqual(t, count, 10)
	})
}

func TestCancelStream(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("cancel existing stream", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		err := sm.RegisterStream("msg-cancel-1", cancel)
		require.NoError(t, err)

		// Verify context is not canceled yet
		select {
		case <-ctx.Done():
			require.Fail(t, "context should not be canceled yet")
		default:
		}

		// Cancel the stream
		err = sm.CancelStream("msg-cancel-1")
		require.NoError(t, err)

		// Verify context is canceled
		select {
		case <-ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			require.Fail(t, "context should be canceled")
		}

		// Verify stream is removed from map
		sm.mu.RLock()
		_, exists := sm.streams["msg-cancel-1"]
		sm.mu.RUnlock()
		require.False(t, exists)
	})

	t.Run("cancel non-existent stream returns error", func(t *testing.T) {
		err := sm.CancelStream("msg-nonexistent")
		require.Error(t, err)
		require.Contains(t, err.Error(), "stream not found")
	})

	t.Run("cancel removes stream from map", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-cancel-2", cancel)
		require.NoError(t, err)

		sm.mu.RLock()
		_, exists := sm.streams["msg-cancel-2"]
		sm.mu.RUnlock()
		require.True(t, exists)

		err = sm.CancelStream("msg-cancel-2")
		require.NoError(t, err)

		sm.mu.RLock()
		_, exists = sm.streams["msg-cancel-2"]
		sm.mu.RUnlock()
		require.False(t, exists)
	})

	t.Run("cancel is idempotent after unregister", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-cancel-3", cancel)
		require.NoError(t, err)

		// Unregister first
		sm.UnregisterStream("msg-cancel-3")

		// Now cancel should fail
		err = sm.CancelStream("msg-cancel-3")
		require.Error(t, err)
		require.Contains(t, err.Error(), "stream not found")
	})
}

func TestUnregisterStream(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("unregister existing stream", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-unregister-1", cancel)
		require.NoError(t, err)

		sm.mu.RLock()
		_, exists := sm.streams["msg-unregister-1"]
		sm.mu.RUnlock()
		require.True(t, exists)

		// Unregister should not panic
		sm.UnregisterStream("msg-unregister-1")

		sm.mu.RLock()
		_, exists = sm.streams["msg-unregister-1"]
		sm.mu.RUnlock()
		require.False(t, exists)
	})

	t.Run("unregister non-existent stream does not panic", func(t *testing.T) {
		// Should not panic
		sm.UnregisterStream("msg-nonexistent-2")
	})

	t.Run("unregister multiple times does not panic", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-unregister-3", cancel)
		require.NoError(t, err)

		sm.UnregisterStream("msg-unregister-3")
		sm.UnregisterStream("msg-unregister-3")
		sm.UnregisterStream("msg-unregister-3")
		// Should not panic
	})

	t.Run("unregister does not cancel context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-unregister-4", cancel)
		require.NoError(t, err)

		// Verify context is not canceled
		select {
		case <-ctx.Done():
			require.Fail(t, "context should not be canceled")
		default:
		}

		sm.UnregisterStream("msg-unregister-4")

		// Context should still not be canceled (unregister doesn't call cancel)
		select {
		case <-ctx.Done():
			require.Fail(t, "context should not be canceled after unregister")
		default:
		}
	})
}

func TestStreamManager_Concurrency(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("concurrent register", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		for i := range numGoroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, cancel := context.WithCancel(context.Background())
				defer cancel()
				_ = sm.RegisterStream("msg-concurrent-"+string(rune(id)), cancel)
			}(i)
		}

		wg.Wait()

		sm.mu.RLock()
		count := len(sm.streams)
		sm.mu.RUnlock()
		require.Equal(t, numGoroutines, count)
	})

	t.Run("concurrent register and unregister", func(t *testing.T) {
		sm2 := &StreamManager{
			streams: make(map[string]context.CancelFunc),
		}

		var wg sync.WaitGroup
		numGoroutines := 50

		// Register streams
		for i := range numGoroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, cancel := context.WithCancel(context.Background())
				defer cancel()
				_ = sm2.RegisterStream("msg-concurrent-2-"+string(rune(id)), cancel)
			}(i)
		}

		wg.Wait()

		// Concurrent unregister
		for i := range numGoroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sm2.UnregisterStream("msg-concurrent-2-" + string(rune(id)))
			}(i)
		}

		wg.Wait()

		sm2.mu.RLock()
		count := len(sm2.streams)
		sm2.mu.RUnlock()
		require.Equal(t, 0, count)
	})

	t.Run("concurrent cancel", func(t *testing.T) {
		sm3 := &StreamManager{
			streams: make(map[string]context.CancelFunc),
		}

		// Register streams first
		for i := range 50 {
			_, cancel := context.WithCancel(context.Background())
			_ = sm3.RegisterStream("msg-concurrent-3-"+string(rune(i)), cancel)
		}

		var wg sync.WaitGroup
		for i := range 50 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_ = sm3.CancelStream("msg-concurrent-3-" + string(rune(id)))
			}(i)
		}

		wg.Wait()

		sm3.mu.RLock()
		count := len(sm3.streams)
		sm3.mu.RUnlock()
		require.Equal(t, 0, count)
	})

	t.Run("concurrent register duplicate", func(t *testing.T) {
		sm4 := &StreamManager{
			streams: make(map[string]context.CancelFunc),
		}

		messageID := "msg-concurrent-4"
		var wg sync.WaitGroup
		numGoroutines := 10
		successCount := 0
		var mu sync.Mutex

		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, cancel := context.WithCancel(context.Background())
				defer cancel()
				err := sm4.RegisterStream(messageID, cancel)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		// Only one should succeed
		require.Equal(t, 1, successCount)

		sm4.mu.RLock()
		_, exists := sm4.streams[messageID]
		sm4.mu.RUnlock()
		require.True(t, exists)
	})
}

func TestStreamManager_EdgeCases(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("empty message ID", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("", cancel)
		require.NoError(t, err)

		err = sm.CancelStream("")
		require.NoError(t, err)
	})

	t.Run("nil cancel function", func(t *testing.T) {
		// Register with nil cancel function
		err := sm.RegisterStream("msg-nil-cancel", nil)
		require.NoError(t, err)

		// Cancel should not panic even if cancel is nil
		err = sm.CancelStream("msg-nil-cancel")
		require.NoError(t, err)
	})

	t.Run("register after cancel", func(t *testing.T) {
		_, cancel1 := context.WithCancel(context.Background())
		defer cancel1()

		err := sm.RegisterStream("msg-after-cancel", cancel1)
		require.NoError(t, err)

		err = sm.CancelStream("msg-after-cancel")
		require.NoError(t, err)

		// Should be able to register again after cancel
		_, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		err = sm.RegisterStream("msg-after-cancel", cancel2)
		require.NoError(t, err)
	})

	t.Run("cancel multiple times", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-cancel-multiple", cancel)
		require.NoError(t, err)

		// First cancel should succeed
		err = sm.CancelStream("msg-cancel-multiple")
		require.NoError(t, err)

		// Second cancel should fail (stream not found)
		err = sm.CancelStream("msg-cancel-multiple")
		require.Error(t, err)
		require.Contains(t, err.Error(), "stream not found")
	})
}

func TestStreamManager_Integration(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("full lifecycle", func(t *testing.T) {
		messageID := "msg-lifecycle"

		// 1. Register
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream(messageID, cancel)
		require.NoError(t, err)

		// 2. Verify registered
		sm.mu.RLock()
		_, exists := sm.streams[messageID]
		sm.mu.RUnlock()
		require.True(t, exists)

		// 3. Cancel
		err = sm.CancelStream(messageID)
		require.NoError(t, err)

		// 4. Verify canceled
		select {
		case <-ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			require.Fail(t, "context should be canceled")
		}

		// 5. Verify removed
		sm.mu.RLock()
		_, exists = sm.streams[messageID]
		sm.mu.RUnlock()
		require.False(t, exists)
	})

	t.Run("register -> unregister -> register again", func(t *testing.T) {
		messageID := "msg-reregister"

		// First register
		ctx1, cancel1 := context.WithCancel(context.Background())
		defer cancel1()
		err := sm.RegisterStream(messageID, cancel1)
		require.NoError(t, err)

		// Unregister
		sm.UnregisterStream(messageID)

		// Register again with different cancel
		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		err = sm.RegisterStream(messageID, cancel2)
		require.NoError(t, err)

		// Verify second cancel works
		err = sm.CancelStream(messageID)
		require.NoError(t, err)

		select {
		case <-ctx2.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			require.Fail(t, "second context should be canceled")
		}

		// First context should not be canceled
		select {
		case <-ctx1.Done():
			require.Fail(t, "first context should not be canceled")
		default:
		}
	})
}

func TestStreamManager_ErrorMessages(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("register duplicate error message", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := sm.RegisterStream("msg-error-1", cancel)
		require.NoError(t, err)

		_, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		err = sm.RegisterStream("msg-error-1", cancel2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "stream already exists")
		require.Contains(t, err.Error(), "msg-error-1")
	})

	t.Run("cancel non-existent error message", func(t *testing.T) {
		err := sm.CancelStream("msg-error-2")
		require.Error(t, err)
		require.Contains(t, err.Error(), "stream not found")
		require.Contains(t, err.Error(), "msg-error-2")
	})
}

func TestStreamManager_ThreadSafety(t *testing.T) {
	sm := &StreamManager{
		streams: make(map[string]context.CancelFunc),
	}

	t.Run("race condition test", func(t *testing.T) {
		messageID := "msg-race"
		var wg sync.WaitGroup

		// Multiple goroutines trying to register the same stream
		for range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, cancel := context.WithCancel(context.Background())
				defer cancel()
				_ = sm.RegisterStream(messageID, cancel)
			}()
		}

		wg.Wait()

		// Only one should succeed
		sm.mu.RLock()
		_, exists := sm.streams[messageID]
		sm.mu.RUnlock()
		require.True(t, exists)

		// Clean up
		sm.UnregisterStream(messageID)
	})

	t.Run("concurrent read and write", func(t *testing.T) {
		sm2 := &StreamManager{
			streams: make(map[string]context.CancelFunc),
		}

		var wg sync.WaitGroup
		done := make(chan struct{})

		// Writer goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				_, cancel := context.WithCancel(context.Background())
				_ = sm2.RegisterStream("msg-rw-"+string(rune(i)), cancel)
				time.Sleep(time.Millisecond)
			}
			close(done)
		}()

		// Reader goroutines
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-done:
						return
					default:
						sm2.mu.RLock()
						_ = len(sm2.streams)
						sm2.mu.RUnlock()
						time.Sleep(time.Millisecond)
					}
				}
			}()
		}

		wg.Wait()
		// Should not panic or deadlock
	})
}
