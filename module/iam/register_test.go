package iam_test

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/forbearing/gst/bootstrap"
	"github.com/forbearing/gst/client"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	"github.com/forbearing/gst/internal/helper"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/module/iam"
	"github.com/stretchr/testify/require"
)

var (
	token = "-"
	port  = 8000

	signupAPI         = fmt.Sprintf("http://localhost:%d/api/signup", port)
	loginAPI          = fmt.Sprintf("http://localhost:%d/api/login", port)
	logoutAPI         = fmt.Sprintf("http://localhost:%d/api/logout", port)
	changepasswordAPI = fmt.Sprintf("http://localhost:%d/api/iam/change-password", port)
	resetpasswordAPI  = fmt.Sprintf("http://localhost:%d/api/iam/reset-password", port)
	accountstatusAPI  = fmt.Sprintf("http://localhost:%d/api/iam/account-status", port)
	userAPI           = fmt.Sprintf("http://localhost:%d/api/iam/users", port)
	groupAPI          = fmt.Sprintf("http://localhost:%d/api/iam/groups", port)
	meAPI             = fmt.Sprintf("http://localhost:%d/api/me", port)
	heartbeatAPI      = fmt.Sprintf("http://localhost:%d/api/heartbeat", port)
	onlineuserAPI     = fmt.Sprintf("http://localhost:%d/api/online-users", port)
	offlineAPI        = fmt.Sprintf("http://localhost:%d/api/offline", port)
)

type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

func init() {
	os.Setenv(config.DATABASE_TYPE, string(config.DBSqlite))
	os.Setenv(config.SQLITE_IS_MEMORY, "true")
	os.Setenv(config.SERVER_PORT, fmt.Sprintf("%d", port))
	os.Setenv(config.REDIS_ENABLE, "true")
	os.Setenv(config.LOGGER_DIR, "./logs")
	os.Setenv(config.AUTH_NONE_EXPIRE_TOKEN, token)

	if err := bootstrap.Bootstrap(); err != nil {
		panic(err)
	}

	go func() {
		iam.Register()

		if err := bootstrap.Run(); err != nil {
			panic(err)
		}
	}()

	for {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			l.Close()
			time.Sleep(1 * time.Second)
			continue
		}
		if errors.Is(err, syscall.EADDRINUSE) {
			break
		}
		panic(err)

	}
}

func TestIAM(t *testing.T) {
	username := "user01"
	oldPassword := "12345678"
	newPassword := "123456789"
	userID := ""

	t.Run("signup", func(t *testing.T) {
		cli, err := client.New(signupAPI)
		require.NoError(t, err)

		resp, err := cli.Create(iam.SignupReq{
			Username:   username,
			Password:   oldPassword,
			RePassword: oldPassword,
		})
		require.NoError(t, err)
		helper.TestResp(t, resp, func(t *testing.T, rsp iam.SignupRsp) {
			// #modeliam.SignupRsp {
			//   +UserID   => "019cbca0-19d4-7971-8be5-65b148027a27" #string
			//   +Username => "user01" #string
			//   +Message  => "User created successfully" #string
			// }
			require.Equal(t, rsp.Username, username)
			require.NotEmpty(t, rsp.UserID)
			require.NotEmpty(t, rsp.Message)
			userID = rsp.UserID
		})
	})

	var sessionID string
	t.Run("login", func(t *testing.T) {
		cli, err := client.New(loginAPI)
		require.NoError(t, err)

		resp, err := cli.Create(iam.LoginReq{
			Username: username,
			Password: oldPassword,
		})
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
			// #*modeliam.LoginRsp {
			//   +SessionID => "019cbca0-1a0b-7a12-8264-4c0525076cd6" #string
			// }
			require.NotEmpty(t, rsp.SessionID)
			sessionID = rsp.SessionID
		})
	})

	t.Run("logout", func(t *testing.T) {
		t.Run("logout", func(t *testing.T) {
			cli, err := client.New(logoutAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(nil)
			require.NoError(t, err)

			helper.TestResp[*iam.LogoutRsp](t, resp, func(t *testing.T, rsp *iam.LogoutRsp) {
				// #*modeliam.LogoutRsp {
				//   +Msg => "logout successful" #string
				// }
				require.NotEmpty(t, rsp.Msg)
			})
		})

		// query user api will failed
		t.Run("user", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			items := make([]*iam.User, 0)
			total := new(int64)
			_, err = cli.List(&items, total)
			require.Error(t, err)
		})

		// login again
		t.Run("login", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: username,
				Password: oldPassword,
			})
			require.NoError(t, err)

			helper.TestResp[*iam.LoginRsp](t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				// #*modeliam.LoginRsp {
				//   +SessionID => "019cbca0-1a47-74fd-ae8a-9a2de4f1bd28" #string
				// }
				require.NotEmpty(t, rsp.SessionID)
				sessionID = rsp.SessionID
			})
		})
	})

	t.Run("user", func(t *testing.T) {
		cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		items := make([]iam.User, 0)
		total := new(int64)
		resp, err := cli.List(&items, total)
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*iam.User]) {
			require.Len(t, rsp.Items, 1)
			u := rsp.Items[0]
			require.NotEmpty(t, u)
			require.Equal(t, u.Username, username)
			require.NotEmpty(t, u.ID)
			require.Equal(t, u.Status, modeliam.UserStatusActive)
			require.Equal(t, u.Type, modeliam.UserTypeRegular)
		})
	})

	t.Run("changepassworhd", func(t *testing.T) {
		// change password
		t.Run("change", func(t *testing.T) {
			cli, err := client.New(changepasswordAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.ChangePasswordReq{
				OldPassword: oldPassword,
				NewPassword: newPassword,
			})
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp *iam.ChangePasswordRsp) {
				// #*modeliam.ChangePasswordRsp {
				//   +Msg => "password changed successfully" #string
				// }
				require.NotEmpty(t, rsp.Msg)
			})
		})

		// login use new password
		t.Run("login", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: username,
				Password: newPassword,
			})
			require.NoError(t, err)

			helper.TestResp[*iam.LoginRsp](t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				// #*modeliam.LoginRsp {
				//   +SessionID => "019cbca0-1ae6-75d8-a63d-1bbaeb31c02b" #string
				// }
				require.NotEmpty(t, rsp.SessionID)
				sessionID = rsp.SessionID
			})
		})

		// list user use new session
		t.Run("user", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			items := make([]iam.User, 0)
			total := new(int64)
			resp, err := cli.List(&items, total)
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*iam.User]) {
				require.Len(t, rsp.Items, 1)
				u := rsp.Items[0]
				require.NotEmpty(t, u)
				require.Equal(t, u.Username, username)
				require.Equal(t, u.Status, modeliam.UserStatusActive)
				require.Equal(t, u.Type, modeliam.UserTypeRegular)
			})
		})
	})

	t.Run("group", func(t *testing.T) {
		cli, err := client.New(groupAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		items := make([]*iam.Group, 0)
		total := new(int64)
		resp, err := cli.List(&items, total)
		require.NoError(t, err)

		helper.TestResp[ListResponse[*iam.Group]](t, resp, func(t *testing.T, rsp ListResponse[*iam.Group]) {
			require.Len(t, rsp.Items, 0)
		})
	})

	t.Run("heartbeat", func(t *testing.T) {
		cli, err := client.New(heartbeatAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		_, err = cli.Create(nil)
		require.NoError(t, err)
	})

	t.Run("me", func(t *testing.T) {
		cli, err := client.New(meAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		empty := new(struct{})
		resp, err := cli.Request(http.MethodGet, empty)
		require.NoError(t, err)

		helper.TestResp(t, resp, func(t *testing.T, rsp iam.MeRsp) {
			// #map[string]interface {} {
			//    user_id => "019cbca3-c442-72f5-bfd5-110077bed415" #string
			//    username => "user01" #string
			//    email => interface {}(nil)
			//    first_name => interface {}(nil)
			//    group => #map[string]interface {} {
			//      path => "" #string
			//      status => "" #string
			//      tenant_id => interface {}(nil)
			//      type => "" #string
			//      id => "" #string
			//      level => 0.000000 #float64
			//      name => "" #string
			//      parent_id => interface {}(nil)
			//   }
			//    last_name => interface {}(nil)
			// }
			require.NotEmpty(t, rsp)
			for k, v := range rsp {
				switch k {
				case "user_id":
					require.NotEmpty(t, v)
				case "username":
					require.NotEmpty(t, v)
				}
			}
		})
	})

	t.Run("onlineuser", func(t *testing.T) {
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
			// godump.Dump(rsp)
			require.Len(t, rsp.Items, 1)
			// ou := rsp.Items[0]
			// require.NotEmpty(t, ou)
			// require.Equal(t, ou.UserID, userID)
			// require.Equal(t, ou.Username, username)
		})
	})

	t.Run("resetpassword", func(t *testing.T) {
		victimName := "victim01"
		victimPass := "87654321"
		resetPass := "resetpass9"
		finalPass := "finalpass9"
		var victimID string
		var victimSessionBeforeReset string

		t.Run("signup_victim", func(t *testing.T) {
			cli, err := client.New(signupAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.SignupReq{
				Username:   victimName,
				Password:   victimPass,
				RePassword: victimPass,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.SignupRsp) {
				require.Equal(t, rsp.Username, victimName)
				require.NotEmpty(t, rsp.UserID)
				victimID = rsp.UserID
			})
		})

		t.Run("forbidden_when_not_superuser", func(t *testing.T) {
			cli, err := client.New(resetpasswordAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			_, err = cli.Create(iam.ResetPasswordReq{
				UserID:      victimID,
				NewPassword: resetPass,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "code:")
		})

		t.Run("victim_login_before_reset", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: victimName,
				Password: victimPass,
			})
			require.NoError(t, err)
			helper.TestResp[*iam.LoginRsp](t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				require.NotEmpty(t, rsp.SessionID)
				victimSessionBeforeReset = rsp.SessionID
			})
		})

		t.Run("promote_actor_superuser", func(t *testing.T) {
			actors := make([]*iam.User, 0)
			require.NoError(t, database.Database[*iam.User](nil).WithLimit(1).WithQuery(&iam.User{Username: username}).List(&actors))
			require.Len(t, actors, 1)
			tru := true
			actors[0].IsSuperuser = &tru
			require.NoError(t, database.Database[*iam.User](nil).Update(actors[0]))
		})

		t.Run("reset_success", func(t *testing.T) {
			cli, err := client.New(resetpasswordAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.ResetPasswordReq{
				UserID:      victimID,
				NewPassword: resetPass,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp *iam.ResetPasswordRsp) {
				// #*modeliam.ResetPasswordRsp {
				//   +Msg => "password reset successfully" #string
				// }
				require.NotEmpty(t, rsp.Msg)
			})
		})

		t.Run("victim_session_invalid_after_reset", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: victimSessionBeforeReset,
			}))
			require.NoError(t, err)

			items := make([]iam.User, 0)
			total := new(int64)
			_, err = cli.List(&items, total)
			require.Error(t, err)
			require.Contains(t, err.Error(), "401")
		})

		var victimSessionAfterReset string
		t.Run("victim_login_after_reset", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: victimName,
				Password: resetPass,
			})
			require.NoError(t, err)
			helper.TestResp[*iam.LoginRsp](t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				require.NotEmpty(t, rsp.SessionID)
				victimSessionAfterReset = rsp.SessionID
			})
		})

		t.Run("must_change_password_blocks_list", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: victimSessionAfterReset,
			}))
			require.NoError(t, err)

			items := make([]iam.User, 0)
			total := new(int64)
			_, err = cli.List(&items, total)
			require.Error(t, err)
			require.Contains(t, err.Error(), "403")
		})

		t.Run("victim_change_password", func(t *testing.T) {
			cli, err := client.New(changepasswordAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: victimSessionAfterReset,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.ChangePasswordReq{
				OldPassword: resetPass,
				NewPassword: finalPass,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp *iam.ChangePasswordRsp) {
				require.NotEmpty(t, rsp.Msg)
			})
		})

		t.Run("victim_list_after_change_password", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: victimSessionAfterReset,
			}))
			require.NoError(t, err)

			items := make([]iam.User, 0)
			total := new(int64)
			resp, err := cli.List(&items, total)
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*iam.User]) {
				require.GreaterOrEqual(t, len(rsp.Items), 2)
			})
		})

		t.Run("demote_actor_superuser", func(t *testing.T) {
			actors := make([]*iam.User, 0)
			require.NoError(t, database.Database[*iam.User](nil).WithLimit(1).WithQuery(&iam.User{Username: username}).List(&actors))
			require.Len(t, actors, 1)
			fal := false
			actors[0].IsSuperuser = &fal
			require.NoError(t, database.Database[*iam.User](nil).Update(actors[0]))
		})
	})

	t.Run("accountstatus", func(t *testing.T) {
		acctVictimName := "acctvic01"
		acctVictimPass := "acctpass11"
		var acctVictimID string
		var acctVictimSessionID string

		t.Run("signup_acctvictim", func(t *testing.T) {
			cli, err := client.New(signupAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.SignupReq{
				Username:   acctVictimName,
				Password:   acctVictimPass,
				RePassword: acctVictimPass,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.SignupRsp) {
				require.Equal(t, rsp.Username, acctVictimName)
				require.NotEmpty(t, rsp.UserID)
				acctVictimID = rsp.UserID
			})
		})

		t.Run("acctvictim_login", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: acctVictimName,
				Password: acctVictimPass,
			})
			require.NoError(t, err)
			helper.TestResp[*iam.LoginRsp](t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				require.NotEmpty(t, rsp.SessionID)
				acctVictimSessionID = rsp.SessionID
			})
		})

		t.Run("forbidden_when_not_superuser", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			_, err = cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatusInactive,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "code:")
		})

		t.Run("promote_actor_superuser", func(t *testing.T) {
			actors := make([]*iam.User, 0)
			require.NoError(t, database.Database[*iam.User](nil).WithLimit(1).WithQuery(&iam.User{Username: username}).List(&actors))
			require.Len(t, actors, 1)
			tru := true
			actors[0].IsSuperuser = &tru
			require.NoError(t, database.Database[*iam.User](nil).Update(actors[0]))
		})

		t.Run("disable_acctvictim", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatusInactive,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.AccountStatusRsp) {
				require.Contains(t, rsp.Msg, "success")
			})
		})

		t.Run("session_invalid_after_disable", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: acctVictimSessionID,
			}))
			require.NoError(t, err)

			items := make([]iam.User, 0)
			total := new(int64)
			_, err = cli.List(&items, total)
			require.Error(t, err)
			require.Contains(t, err.Error(), "401")
		})

		t.Run("login_fails_when_inactive", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			_, err = cli.Create(iam.LoginReq{
				Username: acctVictimName,
				Password: acctVictimPass,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "disabled")
		})

		t.Run("enable_acctvictim", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatusActive,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.AccountStatusRsp) {
				require.Contains(t, rsp.Msg, "success")
			})
		})

		var acctVictimSessionAfterEnable string
		t.Run("login_after_enable", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: acctVictimName,
				Password: acctVictimPass,
			})
			require.NoError(t, err)
			helper.TestResp[*iam.LoginRsp](t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				require.NotEmpty(t, rsp.SessionID)
				acctVictimSessionAfterEnable = rsp.SessionID
			})
		})

		t.Run("invalid_status_rejected", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			_, err = cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatus("not-a-valid-status"),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid")
		})

		t.Run("lock_acctvictim", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatusLocked,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.AccountStatusRsp) {
				require.Contains(t, rsp.Msg, "success")
			})
		})

		t.Run("session_invalid_after_lock", func(t *testing.T) {
			cli, err := client.New(userAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: acctVictimSessionAfterEnable,
			}))
			require.NoError(t, err)

			items := make([]iam.User, 0)
			total := new(int64)
			_, err = cli.List(&items, total)
			require.Error(t, err)
			require.Contains(t, err.Error(), "401")
		})

		t.Run("login_fails_when_locked", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			_, err = cli.Create(iam.LoginReq{
				Username: acctVictimName,
				Password: acctVictimPass,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "locked")
		})

		t.Run("unlock_acctvictim", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatusActive,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.AccountStatusRsp) {
				require.Contains(t, rsp.Msg, "success")
			})
		})

		t.Run("status_unchanged_idempotent", func(t *testing.T) {
			cli, err := client.New(accountstatusAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.AccountStatusReq{
				UserID: acctVictimID,
				Status: modeliam.UserStatusActive,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.AccountStatusRsp) {
				require.Contains(t, rsp.Msg, "unchanged")
			})
		})

		t.Run("demote_actor_superuser", func(t *testing.T) {
			actors := make([]*iam.User, 0)
			require.NoError(t, database.Database[*iam.User](nil).WithLimit(1).WithQuery(&iam.User{Username: username}).List(&actors))
			require.Len(t, actors, 1)
			fal := false
			actors[0].IsSuperuser = &fal
			require.NoError(t, database.Database[*iam.User](nil).Update(actors[0]))
		})
	})

	t.Run("offline", func(t *testing.T) {
		cli, err := client.New(offlineAPI, client.WithCookie(&http.Cookie{
			Name:  "session_id",
			Value: sessionID,
		}))
		require.NoError(t, err)

		_, err = cli.Create(iam.OfflineReq{
			UserID: userID,
		})
		require.NoError(t, err)
	})
}
