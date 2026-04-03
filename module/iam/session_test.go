package iam_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/forbearing/gst/client"
	"github.com/forbearing/gst/internal/helper"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	"github.com/forbearing/gst/module/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/types"
	"github.com/stretchr/testify/require"
)

var (
	sessionsAPI   = fmt.Sprintf("http://localhost:%d/api/iam/sessions", port)
	heartbeatAPI  = fmt.Sprintf("http://localhost:%d/api/iam/session/heartbeat", port)
	onlineuserAPI = fmt.Sprintf("http://localhost:%d/api/online-users", port)
	offlineAPI    = fmt.Sprintf("http://localhost:%d/api/offline", port)
)

type sessionTestAccount struct {
	UserID   string
	Username string
	Password string
}

func requireSessionNotFound(t *testing.T, sessionID string) {
	t.Helper()

	sessionKey := modeliamsession.SessionIDKey(sessionID)
	_, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	require.ErrorIs(t, err, types.ErrEntryNotFound)
}

func requireUserSessionContains(t *testing.T, userID, sessionID string) {
	t.Helper()

	userSessionIDs, err := redis.ZRange(modeliamsession.SessionUserKey(userID), 0, -1)
	require.NoError(t, err)
	require.Contains(t, userSessionIDs, sessionID)
}

func requireUserSessionNotContains(t *testing.T, userID, sessionID string) {
	t.Helper()

	userSessionIDs, err := redis.ZRange(modeliamsession.SessionUserKey(userID), 0, -1)
	require.NoError(t, err)
	require.NotContains(t, userSessionIDs, sessionID)
}

func newSessionTestAccount(t *testing.T) sessionTestAccount {
	t.Helper()

	username := fmt.Sprintf("session_%d", time.Now().UnixNano())
	password := "12345678"

	cli, err := client.New(signupAPI)
	require.NoError(t, err)

	resp, err := cli.Create(iam.SignupReq{
		Username:   username,
		Password:   password,
		RePassword: password,
	})
	require.NoError(t, err)

	account := sessionTestAccount{
		Username: username,
		Password: password,
	}
	helper.TestResp(t, resp, func(t *testing.T, rsp iam.SignupRsp) {
		require.Equal(t, username, rsp.Username)
		require.NotEmpty(t, rsp.UserID)
		require.NotEmpty(t, rsp.Message)
		account.UserID = rsp.UserID
	})

	return account
}

func loginSession(t *testing.T, username, password string) string {
	t.Helper()

	cli, err := client.New(loginAPI)
	require.NoError(t, err)

	resp, err := cli.Create(iam.LoginReq{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)

	sessionID := ""
	helper.TestResp(t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
		require.NotEmpty(t, rsp.SessionID)
		sessionID = rsp.SessionID
	})

	return sessionID
}

func TestSessionHeartbeat(t *testing.T) {
	account := newSessionTestAccount(t)
	sessionID := loginSession(t, account.Username, account.Password)

	sessionKey := modeliamsession.SessionIDKey(sessionID)
	before, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	cli, err := client.New(heartbeatAPI, client.WithCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	}))
	require.NoError(t, err)

	resp, err := cli.Create(nil)
	require.NoError(t, err)

	helper.TestResp[*iam.Heartbeat](t, resp, func(t *testing.T, rsp *iam.Heartbeat) {})

	after, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	require.NoError(t, err)
	require.Equal(t, before.ExpiresAt, after.ExpiresAt)
	require.Equal(t, before.LastSeenAt, after.LastSeenAt)
}

func TestSessionCurrent(t *testing.T) {
	t.Run("get_current_session", func(t *testing.T) {
		account := newSessionTestAccount(t)
		sessionID := loginSession(t, account.Username, account.Password)

		cli, err := client.New(currentAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.Request(http.MethodGet, new(struct{}))
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp iam.CurrentListRsp) {
			require.NotEmpty(t, rsp.Principal.UserID)
			require.Equal(t, account.Username, rsp.Principal.Username)
			require.Equal(t, string(modeliam.UserStatusActive), rsp.Principal.Status)
			require.False(t, rsp.Principal.MustChangePassword)
			require.True(t, rsp.Session.IsCurrent)
			require.Equal(t, sessionID, rsp.Session.ID)
		})
	})
}

func TestSessionList(t *testing.T) {
	t.Run("list_current_user_sessions", func(t *testing.T) {
		account := newSessionTestAccount(t)
		otherSessionID := loginSession(t, account.Username, account.Password)
		currentSessionID := loginSession(t, account.Username, account.Password)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: currentSessionID,
		}))
		require.NoError(t, err)

		items := make([]iam.CurrentSession, 0)
		total := new(int64)
		resp, err := cli.List(&items, total)
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[iam.CurrentSession]) {
			require.Len(t, rsp.Items, 2)
			require.EqualValues(t, 2, rsp.Total)

			sessionMap := make(map[string]iam.CurrentSession, len(rsp.Items))
			for i := range rsp.Items {
				sessionMap[rsp.Items[i].ID] = rsp.Items[i]
			}

			require.Contains(t, sessionMap, currentSessionID)
			require.Contains(t, sessionMap, otherSessionID)
			require.True(t, sessionMap[currentSessionID].IsCurrent)
			require.False(t, sessionMap[otherSessionID].IsCurrent)
		})
	})
}

func TestSessionOnlineUsers(t *testing.T) {
	t.Run("list_online_users", func(t *testing.T) {
		account := newSessionTestAccount(t)
		sessionID := loginSession(t, account.Username, account.Password)

		cli, err := client.New(onlineuserAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		items := make([]*iam.OnlineUser, 0)
		total := new(int64)
		resp, err := cli.List(&items, total)
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*iam.OnlineUser]) {
			require.NotEmpty(t, rsp.Items)
		})
	})
}

func TestSessionDelete(t *testing.T) {
	t.Run("delete_non_current_session", func(t *testing.T) {
		account := newSessionTestAccount(t)
		currentSessionID := loginSession(t, account.Username, account.Password)
		otherSessionID := loginSession(t, account.Username, account.Password)

		requireUserSessionContains(t, account.UserID, currentSessionID)
		requireUserSessionContains(t, account.UserID, otherSessionID)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: currentSessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.Delete(otherSessionID)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.SessionsDeleteRsp) {
			require.Equal(t, iam.SessionsDeleteRsp{}, rsp)
		})

		items := make([]iam.CurrentSession, 0)
		total := new(int64)
		resp, err = cli.List(&items, total)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[iam.CurrentSession]) {
			require.Len(t, rsp.Items, 1)
			require.EqualValues(t, 1, rsp.Total)
			require.Equal(t, currentSessionID, rsp.Items[0].ID)
			require.True(t, rsp.Items[0].IsCurrent)
		})

		requireSessionNotFound(t, otherSessionID)
		requireUserSessionNotContains(t, account.UserID, otherSessionID)
		requireUserSessionContains(t, account.UserID, currentSessionID)
	})

	t.Run("delete_missing_session_is_idempotent", func(t *testing.T) {
		account := newSessionTestAccount(t)
		currentSessionID := loginSession(t, account.Username, account.Password)
		missingSessionID := loginSession(t, account.Username, account.Password)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: currentSessionID,
		}))
		require.NoError(t, err)

		_, err = cli.Delete(missingSessionID)
		require.NoError(t, err)

		resp, err := cli.Delete(missingSessionID)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.SessionsDeleteRsp) {
			require.Equal(t, iam.SessionsDeleteRsp{}, rsp)
		})

		items := make([]iam.CurrentSession, 0)
		total := new(int64)
		resp, err = cli.List(&items, total)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[iam.CurrentSession]) {
			require.Len(t, rsp.Items, 1)
			require.EqualValues(t, 1, rsp.Total)
			require.Equal(t, currentSessionID, rsp.Items[0].ID)
			require.True(t, rsp.Items[0].IsCurrent)
		})
	})

	t.Run("forbidden_when_deleting_other_user_session", func(t *testing.T) {
		attacker := newSessionTestAccount(t)
		attackerSessionID := loginSession(t, attacker.Username, attacker.Password)

		victim := newSessionTestAccount(t)
		victimSessionID := loginSession(t, victim.Username, victim.Password)
		requireUserSessionContains(t, victim.UserID, victimSessionID)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: attackerSessionID,
		}))
		require.NoError(t, err)

		_, err = cli.Delete(victimSessionID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "403")

		requireUserSessionContains(t, victim.UserID, victimSessionID)
	})

	t.Run("delete_current_session", func(t *testing.T) {
		account := newSessionTestAccount(t)
		sessionID := loginSession(t, account.Username, account.Password)
		requireUserSessionContains(t, account.UserID, sessionID)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.Delete(sessionID)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.SessionsDeleteRsp) {
			require.Equal(t, iam.SessionsDeleteRsp{}, rsp)
		})

		requireSessionNotFound(t, sessionID)
		requireUserSessionNotContains(t, account.UserID, sessionID)

		currentCli, err := client.New(currentAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		_, err = currentCli.Request(http.MethodGet, new(struct{}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "401")
	})
}

func TestSessionDeleteOthers(t *testing.T) {
	t.Run("delete_all_other_sessions", func(t *testing.T) {
		account := newSessionTestAccount(t)
		currentSessionID := loginSession(t, account.Username, account.Password)
		otherSessionID1 := loginSession(t, account.Username, account.Password)
		otherSessionID2 := loginSession(t, account.Username, account.Password)

		requireUserSessionContains(t, account.UserID, currentSessionID)
		requireUserSessionContains(t, account.UserID, otherSessionID1)
		requireUserSessionContains(t, account.UserID, otherSessionID2)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: currentSessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.Delete("others")
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.SessionsDeleteRsp) {
			require.Equal(t, iam.SessionsDeleteRsp{}, rsp)
		})

		items := make([]iam.CurrentSession, 0)
		total := new(int64)
		resp, err = cli.List(&items, total)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[iam.CurrentSession]) {
			require.Len(t, rsp.Items, 1)
			require.EqualValues(t, 1, rsp.Total)
			require.Equal(t, currentSessionID, rsp.Items[0].ID)
			require.True(t, rsp.Items[0].IsCurrent)
		})

		requireUserSessionContains(t, account.UserID, currentSessionID)
		requireUserSessionNotContains(t, account.UserID, otherSessionID1)
		requireUserSessionNotContains(t, account.UserID, otherSessionID2)
		requireSessionNotFound(t, otherSessionID1)
		requireSessionNotFound(t, otherSessionID2)
	})

	t.Run("idempotent_when_no_other_sessions", func(t *testing.T) {
		account := newSessionTestAccount(t)
		currentSessionID := loginSession(t, account.Username, account.Password)

		cli, err := client.New(sessionsAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: currentSessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.Delete("others")
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.SessionsDeleteRsp) {
			require.Equal(t, iam.SessionsDeleteRsp{}, rsp)
		})

		items := make([]iam.CurrentSession, 0)
		total := new(int64)
		resp, err = cli.List(&items, total)
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[iam.CurrentSession]) {
			require.Len(t, rsp.Items, 1)
			require.EqualValues(t, 1, rsp.Total)
			require.Equal(t, currentSessionID, rsp.Items[0].ID)
			require.True(t, rsp.Items[0].IsCurrent)
		})
	})
}

func TestSessionOffline(t *testing.T) {
	t.Run("offline_removes_remaining_user_sessions", func(t *testing.T) {
		account := newSessionTestAccount(t)
		staleSessionID := loginSession(t, account.Username, account.Password)
		latestSessionID := loginSession(t, account.Username, account.Password)

		logoutCli, err := client.New(logoutAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: staleSessionID,
		}))
		require.NoError(t, err)

		resp, err := logoutCli.Create(nil)
		require.NoError(t, err)
		helper.TestResp[*iam.LogoutRsp](t, resp, func(t *testing.T, rsp *iam.LogoutRsp) {
			require.NotEmpty(t, rsp.Msg)
		})

		requireSessionNotFound(t, staleSessionID)
		requireUserSessionNotContains(t, account.UserID, staleSessionID)
		requireUserSessionContains(t, account.UserID, latestSessionID)

		cli, err := client.New(offlineAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: latestSessionID,
		}))
		require.NoError(t, err)

		_, err = cli.Create(iam.OfflineReq{
			UserID: account.UserID,
		})
		require.NoError(t, err)

		requireSessionNotFound(t, latestSessionID)
		requireUserSessionNotContains(t, account.UserID, latestSessionID)
	})
}

func TestSessionCurrentDelete(t *testing.T) {
	t.Run("delete_current_session", func(t *testing.T) {
		account := newSessionTestAccount(t)
		sessionID := loginSession(t, account.Username, account.Password)
		requireUserSessionContains(t, account.UserID, sessionID)

		cli, err := client.New(currentAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.Request(http.MethodDelete, new(struct{}))
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.CurrentDeleteRsp) {
			require.Equal(t, iam.CurrentDeleteRsp{}, rsp)
		})

		requireSessionNotFound(t, sessionID)
		requireUserSessionNotContains(t, account.UserID, sessionID)

		_, err = cli.Request(http.MethodGet, new(struct{}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "401")
	})
}
