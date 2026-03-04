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
	"github.com/forbearing/gst/internal/helper"
	modeliam "github.com/forbearing/gst/internal/model/iam"
	"github.com/forbearing/gst/module/iam"
	"github.com/goforj/godump"
	"github.com/stretchr/testify/require"
)

var (
	token = "-"
	port  = 8000

	signupAPI         = fmt.Sprintf("http://localhost:%d/api/signup", port)
	loginAPI          = fmt.Sprintf("http://localhost:%d/api/login", port)
	logoutAPI         = fmt.Sprintf("http://localhost:%d/api/logout", port)
	changepasswordAPI = fmt.Sprintf("http://localhost:%d/api/iam/change-password", port)
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
			godump.Dump(rsp)
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
			godump.Dump(rsp)
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
				godump.Dump(rsp)
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
				godump.Dump(rsp)
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
				godump.Dump(rsp)
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
				godump.Dump(rsp)
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
			// godump.Dump(rsp)
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
			require.Len(t, rsp.Items, 1)
			ou := rsp.Items[0]
			require.NotEmpty(t, ou)
			require.Equal(t, ou.UserID, userID)
			require.Equal(t, ou.Username, username)
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
