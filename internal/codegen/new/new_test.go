//nolint:predeclared
package new

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTeplConfigWritesEssentialConfig(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	require.NoError(t, createTeplConfig("myapp"))

	got, err := os.ReadFile("config.ini.example")
	require.NoError(t, err)
	require.Equal(t, `[app]
name = myapp
description = A Go application built with gst framework

[server]
mode = dev
listen =
port = 8080

[database]
type = sqlite

[sqlite]
path = ./data.db
database = main
is_memory = true
enable = true

[mysql]
host = 127.0.0.1
port = 3306
database =
username = root
password =
charset = utf8mb4
enable = true

[postgres]
host = 127.0.0.1
port = 5432
database =
username = postgres
password =
sslmode = disable
timezone = Asia/Shanghai
enable = true

[redis]
enable = false
addr = 127.0.0.1:6379
db = 0
password =
namespace = myapp
`, string(got))
}

func TestEnsureFileExistsReportsCreatedFiles(t *testing.T) {
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	created, err := EnsureFileExists()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"configx/configx.go",
		"cronjob/cronjob.go",
		"middleware/middleware.go",
		"model/model.go",
		"service/service.go",
		"module/module.go",
		"router/router.go",
		"dao/.gitkeep",
		"provider/.gitkeep",
	}, created)

	created, err = EnsureFileExists()
	require.NoError(t, err)
	require.Empty(t, created)
}
