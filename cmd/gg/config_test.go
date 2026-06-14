package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forbearing/gst/config"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfigDefaultsJSONUsesJSONFormat(t *testing.T) {
	out, err := executeConfigCommandForTest("defaults", "--format", "json")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(strings.TrimSpace(out), "{"))
	require.NotContains(t, out, "[app]")

	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &data))
	require.Contains(t, data, "app")
	require.Contains(t, data, "server")
}

func TestConfigDefaultsYAMLCanSelectSection(t *testing.T) {
	out, err := executeConfigCommandForTest("defaults", "server", "--format", "yaml")
	require.NoError(t, err)
	require.NotContains(t, out, "[server]")

	var data map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &data))
	require.ElementsMatch(t, []string{"server"}, mapKeys(data))

	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "dev", fmt.Sprint(server["mode"]))
}

func TestConfigDefaultsTOMLUsesTOMLFormat(t *testing.T) {
	out, err := executeConfigCommandForTest("defaults", "server", "--format", "toml")
	require.NoError(t, err)
	require.Contains(t, out, "[server]")
	require.NotContains(t, out, "{")

	var data map[string]any
	require.NoError(t, toml.Unmarshal([]byte(out), &data))
	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "dev", fmt.Sprint(server["mode"]))
}

func TestConfigDefaultsRejectsUnknownSection(t *testing.T) {
	_, err := executeConfigCommandForTest("defaults", "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown configuration section")
}

func TestConfigConvertINIToYAML(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "config.ini")
	output := filepath.Join(tmp, "config.yaml")
	writeConfigTestFile(t, input, `[server]
port = 8090
mode = dev

[redis]
enable = true
namespace = myapp
`)

	out, err := executeConfigCommandForTest("convert", input, output)
	require.NoError(t, err)
	require.Empty(t, out)

	raw, err := os.ReadFile(output)
	require.NoError(t, err)

	var data map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &data))
	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	redis, ok := data["redis"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "8090", fmt.Sprint(server["port"]))
	require.Equal(t, "myapp", fmt.Sprint(redis["namespace"]))
}

func TestConfigConvertCanWriteStdoutWithToFormat(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "config.ini")
	writeConfigTestFile(t, input, `[server]
port = 8090
`)

	out, err := executeConfigCommandForTest("convert", input, "--to", "json")
	require.NoError(t, err)

	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &data))
	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "8090", fmt.Sprint(server["port"]))
}

func TestConfigConvertOutputCanBeLoadedByConfigPackage(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "config.ini")
	yamlOutput := filepath.Join(tmp, "config.yaml")
	jsonOutput := filepath.Join(tmp, "config.json")
	tomlOutput := filepath.Join(tmp, "config.toml")
	writeConfigTestFile(t, input, `[server]
port = 8094
mode = local

[redis]
enable = true
namespace = converted
`)

	_, err := executeConfigCommandForTest("convert", input, yamlOutput)
	require.NoError(t, err)
	requireConfigFileCanLoad(t, yamlOutput, 8094, config.Mode("local"), "converted")

	_, err = executeConfigCommandForTest("convert", input, jsonOutput)
	require.NoError(t, err)
	requireConfigFileCanLoad(t, jsonOutput, 8094, config.Mode("local"), "converted")

	_, err = executeConfigCommandForTest("convert", input, tomlOutput)
	require.NoError(t, err)
	requireConfigFileCanLoad(t, tomlOutput, 8094, config.Mode("local"), "converted")
}

func TestConfigConvertTOMLToJSON(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "config.toml")
	writeConfigTestFile(t, input, `[server]
port = 8097
mode = "stg"

[redis]
enable = true
namespace = "tomlconverted"
`)

	out, err := executeConfigCommandForTest("convert", input, "--to", "json")
	require.NoError(t, err)

	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &data))
	server, ok := data["server"].(map[string]any)
	require.True(t, ok)
	redis, ok := data["redis"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "8097", fmt.Sprint(server["port"]))
	require.Equal(t, "tomlconverted", fmt.Sprint(redis["namespace"]))
}

func TestConfigConvertRefusesOverwriteWithoutForce(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "config.ini")
	output := filepath.Join(tmp, "config.yaml")
	writeConfigTestFile(t, input, "[server]\nport = 8090\n")
	writeConfigTestFile(t, output, "existing: true\n")

	_, err := executeConfigCommandForTest("convert", input, output)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	_, err = executeConfigCommandForTest("convert", input, output, "--force")
	require.NoError(t, err)
}

func executeConfigCommandForTest(args ...string) (string, error) {
	cmd := newConfigCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func writeConfigTestFile(t *testing.T, filename, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
	require.NoError(t, os.WriteFile(filename, []byte(content), 0o600))
}

func requireConfigFileCanLoad(t *testing.T, filename string, port int, mode config.Mode, namespace string) {
	t.Helper()

	require.NoError(t, withCleanConfigEnvironment(func() error {
		config.SetConfigFile(filename)
		return config.Init()
	}))
	require.Equal(t, port, config.App.Server.Port)
	require.Equal(t, mode, config.App.Server.Mode)
	require.True(t, config.App.Redis.Enable)
	require.Equal(t, namespace, config.App.Redis.Namespace)
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
