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
	actor := userSignupUser(t, "user_list_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	viewer := userSignupUser(t, "user_list_viewer", "12345678")

	cli := userNewClient(t, actor.SessionID)

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		items := make([]*iam.User, 0)
		total := new(int64)
		_, err := cli.List(&items, total)
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("list_users", func(t *testing.T) {
		items := make([]*iam.User, 0)
		total := new(int64)
		resp, err := cli.List(&items, total)
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*iam.User]) {
			actorItem := userFindByUsername(rsp.Items, actor.Username)
			require.NotNil(t, actorItem)
			require.Equal(t, actor.Username, actorItem.Username)
			require.True(t, actorItem.IsSuperuser != nil && *actorItem.IsSuperuser)

			viewerItem := userFindByUsername(rsp.Items, viewer.Username)
			require.NotNil(t, viewerItem)
			require.Equal(t, modeliamuser.UserStatusActive, viewerItem.Status)
			require.Equal(t, modeliamuser.UserTypeRegular, viewerItem.Type)
		})
	})
}

func TestUserCreate(t *testing.T) {
	actor := userSignupUser(t, "user_create_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	cli := userNewClient(t, actor.SessionID)
	targetUsername := fmt.Sprintf("user_create_target_%d", time.Now().UnixNano())
	targetDisplayName := "Created By Superuser"
	targetEmail := fmt.Sprintf("%s@example.com", targetUsername)

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		_, err := cli.Create(iam.User{
			Username: targetUsername,
			Password: "example-UserCreate-local-01",
		})
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("create_user", func(t *testing.T) {
		resp, err := cli.Create(iam.User{
			Username:    targetUsername,
			Password:    "example-UserCreate-local-01",
			DisplayName: &targetDisplayName,
			Email:       &targetEmail,
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			userCleanupUser(t, targetUsername)
		})

		helper.TestResp(t, resp, func(t *testing.T, rsp iam.User) {
			require.NotEmpty(t, rsp.ID)
			require.Equal(t, targetUsername, rsp.Username)
			require.Empty(t, rsp.Password)
			require.Empty(t, rsp.PasswordHash)
			require.NotNil(t, rsp.DisplayName)
			require.Equal(t, targetDisplayName, *rsp.DisplayName)
			require.NotNil(t, rsp.Email)
			require.Equal(t, targetEmail, *rsp.Email)
		})

		stored := userLoadByUsername(t, targetUsername)
		require.NotNil(t, stored.DisplayName)
		require.Equal(t, targetDisplayName, *stored.DisplayName)
		require.NotNil(t, stored.Email)
		require.Equal(t, targetEmail, *stored.Email)
	})
}

func TestUserGet(t *testing.T) {
	actor := userSignupUser(t, "user_get_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	victim := userSignupUser(t, "user_get_target", "example-UserGet-local-01")

	cli := userNewClient(t, actor.SessionID)

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		_, err := cli.Get(victim.UserID, new(iam.User))
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("get_user", func(t *testing.T) {
		got := new(iam.User)
		resp, err := cli.Get(victim.UserID, got)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, response.CodeSuccess.Code(), resp.Code)
		require.Equal(t, victim.UserID, got.ID)
		require.Equal(t, victim.Username, got.Username)
		require.Equal(t, modeliamuser.UserStatusActive, got.Status)
		require.Equal(t, modeliamuser.UserTypeRegular, got.Type)
		require.Empty(t, got.Password)
		require.Empty(t, got.PasswordHash)
	})
}

func TestUserUpdate(t *testing.T) {
	actor := userSignupUser(t, "user_update_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	victim := userSignupUser(t, "user_update_target", "example-UserUpdate-local-01")
	cli := userNewClient(t, actor.SessionID)
	updatedDisplayName := "Updated By PUT"
	updatedEmail := fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano())

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		_, err := cli.Update(victim.UserID, iam.User{
			Username:    victim.Username,
			Status:      modeliamuser.UserStatusActive,
			Type:        modeliamuser.UserTypeRegular,
			DisplayName: &updatedDisplayName,
			Email:       &updatedEmail,
		})
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("update_user", func(t *testing.T) {
		resp, err := cli.Update(victim.UserID, iam.User{
			Username:    victim.Username,
			Status:      modeliamuser.UserStatusActive,
			Type:        modeliamuser.UserTypeRegular,
			DisplayName: &updatedDisplayName,
			Email:       &updatedEmail,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, response.CodeSuccess.Code(), resp.Code)

		stored := userLoadByID(t, victim.UserID)
		require.NotNil(t, stored.DisplayName)
		require.Equal(t, updatedDisplayName, *stored.DisplayName)
		require.NotNil(t, stored.Email)
		require.Equal(t, updatedEmail, *stored.Email)
	})
}

func TestUserPatch(t *testing.T) {
	actor := userSignupUser(t, "user_patch_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	victim := userSignupUser(t, "user_patch_target", "example-UserPatch-local-01")
	cli := userNewClient(t, actor.SessionID)
	patchedDisplayName := "Patched By PATCH"

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		_, err := cli.Patch(victim.UserID, iam.User{
			DisplayName: &patchedDisplayName,
		})
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("patch_user", func(t *testing.T) {
		resp, err := cli.Patch(victim.UserID, iam.User{
			DisplayName: &patchedDisplayName,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, response.CodeSuccess.Code(), resp.Code)

		stored := userLoadByID(t, victim.UserID)
		require.NotNil(t, stored.DisplayName)
		require.Equal(t, patchedDisplayName, *stored.DisplayName)
	})
}

func TestUserDelete(t *testing.T) {
	actor := userSignupUser(t, "user_delete_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)

	victim := userSignupUser(t, "user_delete_victim", "example-DelVic-local-01")
	victim.SessionID = userLoginUser(t, &victim, victim.Password)

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		cli := userNewClient(t, actor.SessionID)
		_, err := cli.Delete(victim.UserID)
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("delete_user_by_id", func(t *testing.T) {
		cli := userNewClient(t, actor.SessionID)
		_, err := cli.Delete(victim.UserID)
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

	t.Run("forbidden_when_not_superuser", func(t *testing.T) {
		cli := userNewClient(t, actor.SessionID)
		_, err := cli.DeleteMany([]string{victim1.UserID, victim2.UserID})
		userRequireForbidden(t, err)
	})

	t.Run("promote_actor_superuser", func(t *testing.T) {
		userSetSuperuser(t, actor.Username, true)
	})

	t.Run("delete_many_users", func(t *testing.T) {
		cli := userNewClient(t, actor.SessionID)
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

func TestUserSuperuserTargetProtection(t *testing.T) {
	actor := userSignupUser(t, "user_super_target_actor", "12345678")
	actor.SessionID = userLoginUser(t, &actor, actor.Password)
	userSetSuperuser(t, actor.Username, true)

	victim := userSignupUser(t, "user_super_target_victim", "example-UserSuper-local-01")
	userSetSuperuser(t, victim.Username, true)

	cli := userNewClient(t, actor.SessionID)
	blockedDisplayName := "blocked-superuser-update"
	blockedUsername := fmt.Sprintf("user_super_target_create_%d", time.Now().UnixNano())
	superuserEnabled := true

	t.Run("create_superuser_forbidden", func(t *testing.T) {
		_, err := cli.Create(iam.User{
			Username:    blockedUsername,
			Password:    "example-UserSuperCreate-local-01",
			IsSuperuser: &superuserEnabled,
		})
		userRequireForbidden(t, err)
	})

	t.Run("get_superuser_forbidden", func(t *testing.T) {
		_, err := cli.Get(victim.UserID, new(iam.User))
		userRequireForbidden(t, err)
	})

	t.Run("update_superuser_forbidden", func(t *testing.T) {
		_, err := cli.Update(victim.UserID, iam.User{
			Username:    victim.Username,
			Status:      modeliamuser.UserStatusActive,
			Type:        modeliamuser.UserTypeRegular,
			DisplayName: &blockedDisplayName,
		})
		userRequireForbidden(t, err)
	})

	t.Run("patch_superuser_forbidden", func(t *testing.T) {
		_, err := cli.Patch(victim.UserID, iam.User{
			DisplayName: &blockedDisplayName,
		})
		userRequireForbidden(t, err)
	})

	t.Run("delete_superuser_forbidden", func(t *testing.T) {
		_, err := cli.Delete(victim.UserID)
		userRequireForbidden(t, err)
	})

	t.Run("delete_many_superuser_forbidden", func(t *testing.T) {
		_, err := cli.DeleteMany([]string{victim.UserID})
		userRequireForbidden(t, err)
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

func userNewClient(t *testing.T, sessionID string) *client.Client {
	t.Helper()

	cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	}))
	require.NoError(t, err)
	return cli
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

func userLoadByID(t *testing.T, userID string) *iam.User {
	t.Helper()

	user := new(iam.User)
	require.NoError(t, database.Database[*iam.User](nil).Get(user, userID))
	require.NotEmpty(t, user.ID)
	return user
}

func userLoadByUsername(t *testing.T, username string) *iam.User {
	t.Helper()

	users := make([]*iam.User, 0)
	require.NoError(t, database.Database[*iam.User](nil).WithLimit(1).WithQuery(&iam.User{Username: username}).List(&users))
	require.Len(t, users, 1)
	require.NotNil(t, users[0])
	require.NotEmpty(t, users[0].ID)
	return users[0]
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

	cli := userNewClient(t, sessionID)
	items := make([]iam.User, 0)
	total := new(int64)
	_, err := cli.List(&items, total)
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func userRequireForbidden(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	require.Contains(t, err.Error(), "403")
	require.Contains(t, err.Error(), fmt.Sprintf(`"code":%d`, response.CodeForbidden.Code()))
}

func userFindByUsername(items []*iam.User, username string) *iam.User {
	for _, item := range items {
		if item != nil && item.Username == username {
			return item
		}
	}
	return nil
}
