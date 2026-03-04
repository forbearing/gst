package version_test

import (
	"encoding/json"
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
	"github.com/forbearing/gst/module/version"
	"github.com/forbearing/gst/response"
	"github.com/goforj/godump"
	"github.com/stretchr/testify/require"
)

var (
	token = "-"
	port  = 8000

	versionAPI = fmt.Sprintf("http://localhost:%d/api/version", port)
)

func init() {
	os.Setenv(config.DATABASE_TYPE, string(config.DBSqlite))
	os.Setenv(config.SQLITE_IS_MEMORY, "true")
	os.Setenv(config.SERVER_PORT, fmt.Sprintf("%d", port))
	os.Setenv(config.LOGGER_DIR, "./logs")
	os.Setenv(config.AUTH_NONE_EXPIRE_TOKEN, token)
	// Enable audit and sync write before Bootstrap so operationlog test can list logs immediately.
	os.Setenv(config.AUDIT_ENABLE, "true")
	os.Setenv(config.AUDIT_ASYNC_WRITE, "false")

	if err := bootstrap.Bootstrap(); err != nil {
		panic(err)
	}

	go func() {
		version.Register()

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

func testResp[RSP any](t *testing.T, resp *client.Resp, checkFn func(t *testing.T, rsp RSP)) {
	require.NotNil(t, resp)
	require.NotNil(t, resp.Data)
	require.Equal(t, resp.Code, response.CodeSuccess.Code())
	require.Equal(t, resp.Msg, response.CodeSuccess.Msg())
	require.NotEmpty(t, resp.RequestID)
	require.NotEmpty(t, resp.Data)

	var rsp RSP
	require.NoError(t, json.Unmarshal(resp.Data, &rsp))
	if checkFn != nil {
		checkFn(t, rsp)
	}
}

func TestVersion(t *testing.T) {
	cli, err := client.New(versionAPI)
	require.NoError(t, err)

	resp, err := cli.Request(http.MethodGet, nil)
	require.NoError(t, err)

	testResp(t, resp, func(t *testing.T, rsp *version.VersionRsp) {
		godump.Dump(rsp)
		require.NotEmpty(t, rsp)
		require.NotEmpty(t, rsp.BuildTime)
		require.NotEmpty(t, rsp.GoVersion)
		require.NotEmpty(t, rsp.Timestamp)
	})
}
