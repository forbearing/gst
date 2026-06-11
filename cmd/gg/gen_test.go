package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenRunUpdatesRegistrationFilesWhenModelDirHasNoModels(t *testing.T) {
	tests := []struct {
		name  string
		prune bool
	}{
		{name: "without_prune"},
		{name: "with_prune", prune: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEmptyModelProject(t)
			prune = tt.prune

			files := map[string]string{
				filepath.Join(modelDir, "model.go"):     "package model\n\nconst staleModelRegister = \"stale-model-register\"\n",
				filepath.Join(serviceDir, "service.go"): "package service\n\nconst staleServiceRegister = \"stale-service-register\"\n",
				filepath.Join(routerDir, "router.go"):   "package router\n\nconst staleRouterRegister = \"stale-router-register\"\n",
			}
			for filename, content := range files {
				writeTestFile(t, filename, content)
			}

			genRun()

			assertFileNotContains(t, filepath.Join(modelDir, "model.go"), "stale-model-register")
			assertFileNotContains(t, filepath.Join(serviceDir, "service.go"), "stale-service-register")
			assertFileNotContains(t, filepath.Join(routerDir, "router.go"), "stale-router-register")
		})
	}
}

func assertFileNotContains(t *testing.T, filename, text string) {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), text) {
		t.Fatalf("expected %s to not contain %q", filename, text)
	}
}
