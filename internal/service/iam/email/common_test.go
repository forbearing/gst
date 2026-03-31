package serviceiamemail

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamemail "github.com/forbearing/gst/internal/model/iam/email"
	loggerzap "github.com/forbearing/gst/logger/zap"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type testCacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

type testCache[T any] struct {
	items map[string]testCacheEntry[T]
}

func newTestCache[T any]() *testCache[T] {
	return &testCache[T]{items: make(map[string]testCacheEntry[T])}
}

func (c *testCache[T]) Get(key string) (T, error) {
	var zero T
	entry, ok := c.items[key]
	if !ok {
		return zero, types.ErrEntryNotFound
	}
	if !entry.expiresAt.IsZero() && !entry.expiresAt.After(emailNow()) {
		delete(c.items, key)
		return zero, types.ErrEntryNotFound
	}
	return entry.value, nil
}

func (c *testCache[T]) Peek(key string) (T, error) {
	return c.Get(key)
}

func (c *testCache[T]) Set(key string, value T, ttl time.Duration) error {
	entry := testCacheEntry[T]{value: value}
	if ttl > 0 {
		entry.expiresAt = emailNow().Add(ttl)
	}
	c.items[key] = entry
	return nil
}

func (c *testCache[T]) Delete(key string) error {
	if _, ok := c.items[key]; !ok {
		return types.ErrEntryNotFound
	}
	delete(c.items, key)
	return nil
}

func (c *testCache[T]) Exists(key string) bool {
	_, err := c.Get(key)
	return err == nil
}

func (c *testCache[T]) Len() int {
	return len(c.items)
}

func (c *testCache[T]) Clear() {
	clear(c.items)
}

func (c *testCache[T]) WithContext(context.Context) types.Cache[T] {
	return c
}

type testEmailSender struct {
	last emailDelivery
}

func (s *testEmailSender) Send(_ context.Context, delivery emailDelivery) error {
	s.last = delivery
	return nil
}

func TestIssueLoadConsumeEmailFlow(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{1}, 64)))
	t.Cleanup(restore)

	token, issued, err := issueEmailFlow(context.Background(), iamEmailFlowKindVerification, iamEmailFlowState{
		Email:    " USER@Example.COM ",
		Metadata: map[string]any{"source": "signup"},
	}, 0)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, iamEmailFlowKindVerification, issued.Kind)
	require.Equal(t, "user@example.com", issued.Email)
	require.Equal(t, now, issued.IssuedAt)
	require.Equal(t, now.Add(24*time.Hour), issued.ExpiresAt)

	loaded, err := loadEmailFlow(context.Background(), iamEmailFlowKindVerification, token)
	require.NoError(t, err)
	require.Equal(t, issued, loaded)

	consumed, err := consumeEmailFlow(context.Background(), iamEmailFlowKindVerification, token)
	require.NoError(t, err)
	require.Equal(t, issued, consumed)

	_, err = loadEmailFlow(context.Background(), iamEmailFlowKindVerification, token)
	require.ErrorIs(t, err, errEmailFlowNotFound)
}

func TestLoadEmailFlowExpired(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{2}, 64)))
	t.Cleanup(restore)

	token, err := newEmailFlowToken()
	require.NoError(t, err)
	err = flowCache.Set(emailFlowKey(iamEmailFlowKindPasswordReset, token), iamEmailFlowState{
		Kind:      iamEmailFlowKindPasswordReset,
		Email:     "user@example.com",
		IssuedAt:  now.Add(-2 * time.Minute),
		ExpiresAt: now.Add(-1 * time.Minute),
	}, 0)
	require.NoError(t, err)

	emailNow = func() time.Time { return now.Add(2 * time.Minute) }

	_, err = loadEmailFlow(context.Background(), iamEmailFlowKindPasswordReset, token)
	require.ErrorIs(t, err, errEmailFlowExpired)
}

func TestReserveEmailThrottle(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{3}, 64)))
	t.Cleanup(restore)

	wait, err := reserveEmailThrottle(context.Background(), iamEmailFlowKindVerification, emailThrottleRequest, "USER@example.com", time.Minute)
	require.NoError(t, err)
	require.Zero(t, wait)

	wait, err = reserveEmailThrottle(context.Background(), iamEmailFlowKindVerification, emailThrottleRequest, "user@example.com", time.Minute)
	require.ErrorIs(t, err, errEmailFlowThrottled)
	require.Greater(t, wait, time.Duration(0))

	emailNow = func() time.Time { return now.Add(2 * time.Minute) }

	wait, err = reserveEmailThrottle(context.Background(), iamEmailFlowKindVerification, emailThrottleRequest, "user@example.com", time.Minute)
	require.NoError(t, err)
	require.Zero(t, wait)
}

func TestDispatchEmail(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{4}, 64)))
	t.Cleanup(restore)

	sender := new(testEmailSender)
	setEmailSender(sender)

	err := dispatchEmail(context.Background(), emailDelivery{To: "  USER@Example.COM  ", Subject: "Verify"})
	require.NoError(t, err)
	require.Equal(t, "user@example.com", sender.last.To)
	require.Equal(t, "Verify", sender.last.Subject)

	err = dispatchEmail(context.Background(), emailDelivery{})
	require.EqualError(t, err, "email recipient is required")
}

func TestPublicAcceptedMessage(t *testing.T) {
	require.Equal(t, "If the email is eligible, a verification message will be sent shortly.", publicAcceptedMessage(iamEmailFlowKindVerification))
	require.Equal(t, "If the email is eligible, a password reset message will be sent shortly.", publicAcceptedMessage(iamEmailFlowKindPasswordReset))
}

func stubEmailGlobals(flowCache types.Cache[iamEmailFlowState], throttleCache types.Cache[emailThrottleRecord], now time.Time, reader *bytes.Reader) func() {
	previousFlowCache := emailFlowCache
	previousThrottleCache := emailThrottleCache
	previousNow := emailNow
	previousReader := emailRandomReader
	previousSender := activeEmailSender

	emailFlowCache = func() types.Cache[iamEmailFlowState] { return flowCache }
	emailThrottleCache = func() types.Cache[emailThrottleRecord] { return throttleCache }
	emailNow = func() time.Time { return now }
	emailRandomReader = reader
	activeEmailSender = noopEmailSender{}

	return func() {
		emailFlowCache = previousFlowCache
		emailThrottleCache = previousThrottleCache
		emailNow = previousNow
		emailRandomReader = previousReader
		activeEmailSender = previousSender
	}
}

func TestInvalidKind(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{5}, 64)))
	t.Cleanup(restore)

	_, _, err := issueEmailFlow(context.Background(), iamEmailFlowKind("unknown"), iamEmailFlowState{}, 0)
	require.ErrorIs(t, err, errEmailFlowKindInvalid)

	_, err = reserveEmailThrottle(context.Background(), iamEmailFlowKind("unknown"), emailThrottleRequest, "user@example.com", time.Minute)
	require.ErrorIs(t, err, errEmailFlowKindInvalid)
}

func TestMissingTokenReturnsNotFound(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 13, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{6}, 64)))
	t.Cleanup(restore)

	_, err := loadEmailFlow(context.Background(), iamEmailFlowKindVerification, " ")
	require.True(t, errors.Is(err, errEmailFlowNotFound))
}

func TestPasswordResetRequestCreate(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{7}, 64)))
	t.Cleanup(restore)

	sender := new(testEmailSender)
	setEmailSender(sender)

	previousLookup := passwordResetLookupUserByEmail
	passwordResetLookupUserByEmail = func(_ *types.ServiceContext, email string) (*modeliam.User, error) {
		require.Equal(t, "user@example.com", email)
		emailCopy := "user@example.com"
		return &modeliam.User{
			Base:   model.Base{ID: "user-1"},
			Status: modeliam.UserStatusActive,
			Email:  &emailCopy,
		}, nil
	}
	t.Cleanup(func() {
		passwordResetLookupUserByEmail = previousLookup
	})

	svc := &PasswordResetRequestService{}
	svc.Logger = loggerzap.New("")
	ctx := new(types.ServiceContext)
	ctx.SetPhase(consts.PHASE_CREATE)

	rsp, err := svc.Create(ctx, &modeliamemail.PasswordResetRequestReq{Email: " USER@example.com "})
	require.NoError(t, err)
	require.Equal(t, publicAcceptedMessage(iamEmailFlowKindPasswordReset), rsp.Msg)
	require.Equal(t, "user@example.com", sender.last.To)
	require.Equal(t, "Password reset", sender.last.Subject)
	require.NotEmpty(t, sender.last.Data["token"])
	require.Equal(t, "user-1", sender.last.Data["user_id"])
	require.Equal(t, 1, flowCache.Len())
}

func TestPasswordResetRequestCreateUnknownUser(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 15, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{8}, 64)))
	t.Cleanup(restore)

	sender := new(testEmailSender)
	setEmailSender(sender)

	previousLookup := passwordResetLookupUserByEmail
	passwordResetLookupUserByEmail = func(_ *types.ServiceContext, _ string) (*modeliam.User, error) { return nil, nil }
	t.Cleanup(func() {
		passwordResetLookupUserByEmail = previousLookup
	})

	svc := &PasswordResetRequestService{}
	svc.Logger = loggerzap.New("")
	ctx := new(types.ServiceContext)
	ctx.SetPhase(consts.PHASE_CREATE)

	rsp, err := svc.Create(ctx, &modeliamemail.PasswordResetRequestReq{Email: "user@example.com"})
	require.NoError(t, err)
	require.Equal(t, publicAcceptedMessage(iamEmailFlowKindPasswordReset), rsp.Msg)
	require.Equal(t, 0, flowCache.Len())
	require.Equal(t, "", sender.last.To)
}

func TestPasswordResetConfirmCreate(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 16, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{9}, 64)))
	t.Cleanup(restore)

	token, err := newEmailFlowToken()
	require.NoError(t, err)
	err = flowCache.Set(emailFlowKey(iamEmailFlowKindPasswordReset, token), iamEmailFlowState{
		Kind:      iamEmailFlowKindPasswordReset,
		UserID:    "user-2",
		Email:     "user@example.com",
		IssuedAt:  now,
		ExpiresAt: now.Add(30 * time.Minute),
	}, 30*time.Minute)
	require.NoError(t, err)

	emailCopy := "user@example.com"
	user := &modeliam.User{
		Base:               model.Base{ID: "user-2"},
		Email:              &emailCopy,
		PasswordHash:       "old-hash",
		MustChangePassword: true,
	}

	previousLoad := passwordResetLoadUserByID
	previousUpdate := passwordResetUpdateUser
	previousInvalidate := passwordResetInvalidateSessions
	passwordResetLoadUserByID = func(_ *types.ServiceContext, userID string) (*modeliam.User, error) {
		require.Equal(t, "user-2", userID)
		return user, nil
	}
	passwordResetUpdateUser = func(_ *types.ServiceContext, updated *modeliam.User) error {
		user = updated
		return nil
	}
	var invalidated string
	passwordResetInvalidateSessions = func(userID string) { invalidated = userID }
	t.Cleanup(func() {
		passwordResetLoadUserByID = previousLoad
		passwordResetUpdateUser = previousUpdate
		passwordResetInvalidateSessions = previousInvalidate
	})

	svc := &PasswordResetConfirmService{}
	svc.Logger = loggerzap.New("")
	ctx := new(types.ServiceContext)
	ctx.SetPhase(consts.PHASE_CREATE)

	rsp, err := svc.Create(ctx, &modeliamemail.PasswordResetConfirmReq{
		Token:       token,
		NewPassword: "new-password-123",
	})
	require.NoError(t, err)
	require.True(t, rsp.Reset)
	require.Equal(t, "password reset successfully", rsp.Msg)
	require.Equal(t, "user-2", invalidated)
	require.False(t, user.MustChangePassword)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("new-password-123")))
	_, err = loadEmailFlow(context.Background(), iamEmailFlowKindPasswordReset, token)
	require.ErrorIs(t, err, errEmailFlowNotFound)
}

func TestPasswordResetConfirmCreateInvalidToken(t *testing.T) {
	flowCache := newTestCache[iamEmailFlowState]()
	throttleCache := newTestCache[emailThrottleRecord]()
	now := time.Date(2026, 3, 31, 17, 0, 0, 0, time.UTC)
	restore := stubEmailGlobals(flowCache, throttleCache, now, bytes.NewReader(bytes.Repeat([]byte{10}, 64)))
	t.Cleanup(restore)

	svc := &PasswordResetConfirmService{}
	svc.Logger = loggerzap.New("")
	ctx := new(types.ServiceContext)
	ctx.SetPhase(consts.PHASE_CREATE)

	rsp, err := svc.Create(ctx, &modeliamemail.PasswordResetConfirmReq{
		Token:       "missing",
		NewPassword: "new-password-123",
	})
	require.NoError(t, err)
	require.False(t, rsp.Reset)
	require.Equal(t, "invalid or expired password reset token", rsp.Msg)
}
