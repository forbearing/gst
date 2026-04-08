package iam_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/forbearing/gst/client"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/internal/helper"
	modeliamsession "github.com/forbearing/gst/internal/model/iam/session"
	modeliamuser "github.com/forbearing/gst/internal/model/iam/user"
	"github.com/forbearing/gst/module/iam"
	"github.com/forbearing/gst/provider/redis"
	"github.com/forbearing/gst/response"
	"github.com/forbearing/gst/types"
	"github.com/stretchr/testify/require"
)

type userTestAccount struct {
	UserID    string
	Username  string
	Password  string
	SessionID string
}

func TestUserList(t *testing.T) {
	user := userSignupUser(t, "user_list", "12345678")
	user.SessionID = userLoginUser(t, &user, user.Password)

	cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
		Name:  "session_id",
		Value: user.SessionID,
	}))
	require.NoError(t, err)

	items := make([]iam.User, 0)
	total := new(int64)
	resp, err := cli.List(&items, total)
	require.NoError(t, err)

	helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*iam.User]) {
		target := userFindByUsername(rsp.Items, user.Username)
		require.NotNil(t, target)
		require.NotEmpty(t, target.ID)
		require.Equal(t, user.Username, target.Username)
		require.Equal(t, modeliamuser.UserStatusActive, target.Status)
		require.Equal(t, modeliamuser.UserTypeRegular, target.Type)
	})
}

func TestUserDelete(t *testing.T) {
	actor := userSignupUser(t, "user_delete_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	victim := userSignupUser(t, "user_delete_victim", "example-DelVic-local-01")
	victim.SessionID = userLoginUser(t, &victim, victim.Password)

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("delete_user_by_id", func(t *testing.T) {
		cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: actor.SessionID,
		}))
		require.NoError(t, err)

		_, err = cli.Delete(victim.UserID)
		require.NoError(t, err)
		userRequireDeleted(t, victim.Username)
	})

	t.Run("session_invalid_after_delete", func(t *testing.T) {
		userRequireSessionNotFound(t, victim.SessionID)
		userRequireUserSessionNotContains(t, victim.UserID, victim.SessionID)
		userRequireListUnauthorized(t, victim.SessionID)
	})

	t.Run("demote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, false)
	})
}

func TestUserDeleteMany(t *testing.T) {
	actor := userSignupUser(t, "user_delete_many_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	victim1 := userSignupUser(t, "user_delete_many_victim1", "example-DelMany-local-01")
	victim1.SessionID = userLoginUser(t, &victim1, victim1.Password)

	victim2 := userSignupUser(t, "user_delete_many_victim2", "example-DelMany-local-02")
	victim2.SessionID = userLoginUser(t, &victim2, victim2.Password)

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("delete_many_users", func(t *testing.T) {
		cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: actor.SessionID,
		}))
		require.NoError(t, err)

		resp, err := cli.DeleteMany([]string{victim1.UserID, victim2.UserID})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, response.CodeSuccess.Code(), resp.Code)

		userRequireDeleted(t, victim1.Username)
		userRequireDeleted(t, victim2.Username)
	})

	t.Run("sessions_invalid_after_delete_many", func(t *testing.T) {
		userRequireSessionNotFound(t, victim1.SessionID)
		userRequireUserSessionNotContains(t, victim1.UserID, victim1.SessionID)
		userRequireListUnauthorized(t, victim1.SessionID)

		userRequireSessionNotFound(t, victim2.SessionID)
		userRequireUserSessionNotContains(t, victim2.UserID, victim2.SessionID)
		userRequireListUnauthorized(t, victim2.SessionID)
	})

	t.Run("demote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, false)
	})
}

func userSignupUser(t *testing.T, prefix, password string) userTestAccount {
	t.Helper()

	user := userTestAccount{
		Username: fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()),
		Password: password,
	}

	cli, err := client.New(signupAPI)
	require.NoError(t, err)

	resp, err := cli.Create(iam.SignupReq{
		Username:   user.Username,
		Password:   user.Password,
		RePassword: user.Password,
	})
	require.NoError(t, err)

	helper.TestResp(t, resp, func(t *testing.T, rsp iam.SignupRsp) {
		require.Equal(t, user.Username, rsp.Username)
		require.NotEmpty(t, rsp.UserID)
		require.NotEmpty(t, rsp.Message)
		user.UserID = rsp.UserID
	})

	t.Cleanup(func() {
		userCleanupUser(t, user.Username)
	})

	return user
}

func userCleanupUser(t *testing.T, username string) {
	t.Helper()

	users := make([]*iam.User, 0)
	require.NoError(t, database.Database[*iam.User](nil).WithQuery(&iam.User{Username: username}).List(&users))
	if len(users) == 0 {
		return
	}

	require.NoError(t, database.Database[*iam.User](nil).Delete(users...))
}

func userLoginUser(t *testing.T, user *userTestAccount, password string) string {
	t.Helper()

	cli, err := client.New(loginAPI)
	require.NoError(t, err)

	resp, err := cli.Create(iam.LoginReq{
		Username: user.Username,
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

func userSetSuperuser(t *testing.T, username string, enabled bool) {
	t.Helper()

	users := make([]*iam.User, 0)
	require.NoError(t, database.Database[*iam.User](nil).WithLimit(1).WithQuery(&iam.User{Username: username}).List(&users))
	require.Len(t, users, 1)

	users[0].IsSuperuser = &enabled
	require.NoError(t, database.Database[*iam.User](nil).Update(users[0]))
}

func userRequireDeleted(t *testing.T, username string) {
	t.Helper()

	users := make([]*iam.User, 0)
	require.NoError(t, database.Database[*iam.User](nil).WithQuery(&iam.User{Username: username}).List(&users))
	require.Len(t, users, 0)
}

func userRequireSessionNotFound(t *testing.T, sessionID string) {
	t.Helper()

	sessionKey := modeliamsession.SessionIDKey(sessionID)
	_, err := redis.Cache[modeliamsession.Session]().Get(sessionKey)
	require.ErrorIs(t, err, types.ErrEntryNotFound)
}

func userRequireUserSessionNotContains(t *testing.T, userID, sessionID string) {
	t.Helper()

	userSessionIDs, err := redis.ZRange(modeliamsession.SessionUserKey(userID), 0, -1)
	require.NoError(t, err)
	require.NotContains(t, userSessionIDs, sessionID)
}

func userRequireListUnauthorized(t *testing.T, sessionID string) {
	t.Helper()

	cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	}))
	require.NoError(t, err)

	items := make([]iam.User, 0)
	total := new(int64)
	_, err = cli.List(&items, total)
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func userFindByUsername(items []*iam.User, username string) *iam.User {
	for _, item := range items {
		if item != nil && item.Username == username {
			return item
		}
	}
	return nil
}
