package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneRunPrunesServiceFilesWhenModelDirHasNoModels(t *testing.T) {
	setupEmptyModelProject(t)
	confirmPruneDeletion(t)

	oldServiceFile := filepath.Join(serviceDir, "user", "create.go")
	writeTestFile(t, oldServiceFile, "package user\n\ntype Creator struct{}\n")

	pruneRun()

	if fileExists(oldServiceFile) {
		t.Fatalf("expected old service file %s to be pruned", oldServiceFile)
	}
}

func TestGenRunPrunesServiceFilesWhenModelDirHasNoModels(t *testing.T) {
	setupEmptyModelProject(t)
	confirmPruneDeletion(t)
	prune = true

	oldServiceFile := filepath.Join(serviceDir, "user", "create.go")
	writeTestFile(t, oldServiceFile, "package user\n\ntype Creator struct{}\n")

	genRun()

	if fileExists(oldServiceFile) {
		t.Fatalf("expected old service file %s to be pruned", oldServiceFile)
	}
}

func TestPruneRunRemovesDeepEmptyServiceDirectories(t *testing.T) {
	setupEmptyModelProject(t)
	confirmPruneDeletion(t)

	oldServiceFile := filepath.Join(serviceDir, "config", "namespace", "app", "env", "item", "create.go")
	writeTestFile(t, oldServiceFile, "package item\n\ntype Creator struct{}\n")

	pruneRun()

	emptyServiceDir := filepath.Join(serviceDir, "config")
	if fileExists(emptyServiceDir) {
		t.Fatalf("expected empty service directory %s to be removed", emptyServiceDir)
	}
}

func setupEmptyModelProject(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	oldModelDir := modelDir
	oldServiceDir := serviceDir
	oldRouterDir := routerDir
	oldDAODir := daoDir
	oldExcludes := excludes
	oldModule := module
	oldPrune := prune
	oldGGConfig := ggConfig

	t.Cleanup(func() {
		modelDir = oldModelDir
		serviceDir = oldServiceDir
		routerDir = oldRouterDir
		daoDir = oldDAODir
		excludes = oldExcludes
		module = oldModule
		prune = oldPrune
		ggConfig = oldGGConfig
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})

	t.Chdir(t.TempDir())

	modelDir = "model"
	serviceDir = "service"
	routerDir = "router"
	daoDir = "dao"
	excludes = nil
	module = "example.com/app"
	prune = false
	ggConfig = nil

	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func confirmPruneDeletion(t *testing.T) {
	t.Helper()

	stdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = stdin
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func writeTestFile(t *testing.T, filename, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
