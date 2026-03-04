package version_test

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
	"github.com/forbearing/gst/module/version"
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

func TestVersion(t *testing.T) {
	cli, err := client.New(versionAPI)
	require.NoError(t, err)

	resp, err := cli.Request(http.MethodGet, nil)
	require.NoError(t, err)

	helper.TestResp(t, resp, func(t *testing.T, rsp *version.VersionRsp) {
		godump.Dump(rsp)
		require.NotEmpty(t, rsp)
		require.NotEmpty(t, rsp.BuildTime)
		require.NotEmpty(t, rsp.GoVersion)
		require.NotEmpty(t, rsp.Timestamp)
	})
}
