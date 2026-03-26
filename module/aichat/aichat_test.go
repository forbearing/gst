package aichat_test

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/forbearing/gst/bootstrap"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/module/aichat"
)

var (
	token = "-"
	// Use a port distinct from other module tests (e.g. module/iam uses 8000). Parallel `go test`
	// runs separate packages in separate processes; sharing 8000 makes this package's init() exit
	// on EADDRINUSE while another package's server is bound, so requests hit the wrong router (404).
	port = 18080
	addr = fmt.Sprintf("http://localhost:%d/api", port)
)

func init() {
	os.Setenv(config.DATABASE_TYPE, string(config.DBMySQL))
	os.Setenv(config.SERVER_PORT, fmt.Sprintf("%d", port))
	os.Setenv(config.LOGGER_DIR, "/tmp/aichat")
	os.Setenv(config.AUTH_NONE_EXPIRE_TOKEN, token)
	os.Setenv(config.MYSQL_USERNAME, "aichat")
	os.Setenv(config.MYSQL_PASSWORD, "aichat")
	os.Setenv(config.MYSQL_DATABASE, "aichat")
	os.Setenv(config.MYSQL_HOST, "127.0.0.1")
	os.Setenv(config.MYSQL_PORT, "3307")

	// Improve the server timeout for non-stream chat.
	os.Setenv(config.SERVER_READ_TIMEOUT, "3m")
	os.Setenv(config.SERVER_WRITE_TIMEOUT, "3m")

	if err := bootstrap.Bootstrap(); err != nil {
		panic(err)
	}

	go func() {
		aichat.Register()

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
