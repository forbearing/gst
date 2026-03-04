package logmgmt_test

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
	modellogmgmt "github.com/forbearing/gst/internal/model/logmgmt"
	"github.com/forbearing/gst/module/iam"
	"github.com/forbearing/gst/module/logmgmt"
	"github.com/forbearing/gst/types/consts"
	"github.com/goforj/godump"
	"github.com/stretchr/testify/require"
)

var (
	token = "-"
	port  = 8000

	signupAPI       = fmt.Sprintf("http://localhost:%d/api/signup", port)
	loginAPI        = fmt.Sprintf("http://localhost:%d/api/login", port)
	logoutAPI       = fmt.Sprintf("http://localhost:%d/api/logout", port)
	loginlogAPI     = fmt.Sprintf("http://localhost:%d/api/log/loginlog", port)
	operationlogAPI = fmt.Sprintf("http://localhost:%d/api/log/operationlog", port)
	groupAPI        = fmt.Sprintf("http://localhost:%d/api/iam/groups", port)
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
	// Enable audit and sync write before Bootstrap so operationlog test can list logs immediately.
	os.Setenv(config.AUDIT_ENABLE, "true")
	os.Setenv(config.AUDIT_ASYNC_WRITE, "false")

	if err := bootstrap.Bootstrap(); err != nil {
		panic(err)
	}

	go func() {
		iam.Register()
		logmgmt.Register()

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

func TestLogmgmt(t *testing.T) {
	username := "user01"
	password := "12345678"
	userID := ""
	var sessionID string

	t.Run("loginlog", func(t *testing.T) {
		// signup a user
		t.Run("signup", func(t *testing.T) {
			cli, err := client.New(signupAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.SignupReq{
				Username:   username,
				Password:   password,
				RePassword: password,
			})
			require.NoError(t, err)
			helper.TestResp(t, resp, func(t *testing.T, rsp iam.SignupRsp) {
				godump.Dump(rsp)
				require.Equal(t, rsp.Username, username)
				require.NotEmpty(t, rsp.UserID)
				userID = rsp.UserID
				require.NotEmpty(t, rsp.Message)
			})
		})

		// user login
		t.Run("login1", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: username,
				Password: password,
			})
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				godump.Dump(rsp)
				require.NotEmpty(t, rsp.SessionID)
				sessionID = rsp.SessionID
			})
		})

		// check the login log count is 1
		t.Run("loginlog1", func(t *testing.T) {
			cli, err := client.New(loginlogAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			items := make([]*logmgmt.LoginLog, 0)
			total := new(int64)
			resp, err := cli.List(&items, total)
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*logmgmt.LoginLog]) {
				require.Len(t, rsp.Items, 1)
				l := rsp.Items[0]
				require.Equal(t, l.UserID, userID)
				require.Equal(t, l.Username, username)
				require.Equal(t, string(l.Status), modellogmgmt.LoginStatusSuccess)
			})
		})

		// logout
		t.Run("logout", func(t *testing.T) {
			t.Run("logout", func(t *testing.T) {
				cli, err := client.New(logoutAPI, client.WithCookie(&http.Cookie{
					Name:  "session_id",
					Value: sessionID,
				}))
				require.NoError(t, err)

				resp, err := cli.Create(nil)
				require.NoError(t, err)

				helper.TestResp(t, resp, func(t *testing.T, rsp *iam.LogoutRsp) {
					godump.Dump(rsp)
				})
			})
		})

		// login again to query the login log
		t.Run("login2", func(t *testing.T) {
			cli, err := client.New(loginAPI)
			require.NoError(t, err)

			resp, err := cli.Create(iam.LoginReq{
				Username: username,
				Password: password,
			})
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp *iam.LoginRsp) {
				godump.Dump(rsp)
				require.NotEmpty(t, rsp.SessionID)
				sessionID = rsp.SessionID
			})
		})

		// check the login log count is 2
		t.Run("loginlog2", func(t *testing.T) {
			cli, err := client.New(loginlogAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			items := make([]*logmgmt.LoginLog, 0)
			total := new(int64)
			resp, err := cli.List(&items, total)
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*logmgmt.LoginLog]) {
				require.Len(t, rsp.Items, 3)
				l1, l2, l3 := rsp.Items[0], rsp.Items[1], rsp.Items[2]

				require.Equal(t, l1.UserID, userID)
				require.Equal(t, l1.Username, username)
				require.Equal(t, string(l1.Status), modellogmgmt.LoginStatusSuccess)

				require.Equal(t, l2.UserID, userID)
				require.Equal(t, l2.Username, username)
				require.Equal(t, string(l2.Status), modellogmgmt.LoginStatusLogout)

				require.Equal(t, l3.UserID, userID)
				require.Equal(t, l3.Username, username)
				require.Equal(t, string(l3.Status), modellogmgmt.LoginStatusSuccess)
			})
		})
	})

	t.Run("operationlog", func(t *testing.T) {
		t.Run("operationlog1", func(t *testing.T) {
			cli, err := client.New(operationlogAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			items := make([]*logmgmt.OperationLog, 0)
			total := new(int64)

			resp, err := cli.List(&items, total)
			require.NoError(t, err)

			// operation log count is 0
			helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*logmgmt.OperationLog]) {
				require.Len(t, rsp.Items, 0)
			})
		})

		t.Run("create-group", func(t *testing.T) {
			cli, err := client.New(groupAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			resp, err := cli.Create(iam.Group{
				Name: "g1",
			})
			require.NoError(t, err)

			helper.TestResp(t, resp, func(t *testing.T, rsp *iam.Group) {
				require.NotNil(t, rsp)
				require.Equal(t, rsp.Name, "g1")
			})
		})

		t.Run("operationlog2", func(t *testing.T) {
			cli, err := client.New(operationlogAPI, client.WithCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			}))
			require.NoError(t, err)

			items := make([]*logmgmt.OperationLog, 0)
			total := new(int64)

			resp, err := cli.List(&items, total)
			require.NoError(t, err)

			// operation log count is 1
			helper.TestResp(t, resp, func(t *testing.T, rsp ListResponse[*logmgmt.OperationLog]) {
				require.Len(t, rsp.Items, 1)
				l := rsp.Items[0]
				require.NotNil(t, l)
				require.Equal(t, l.User, username)
				require.Equal(t, l.OP, consts.OP_CREATE)
				require.Equal(t, l.Table, "groups")
				require.Equal(t, l.Model, "Group")
			})
		})
	})
}
