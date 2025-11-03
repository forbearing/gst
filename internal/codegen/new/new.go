//nolint:predeclared
package new

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/fatih/color"
	"github.com/forbearing/gst/config"
)

// ============================================================
// 文件模板映射
// ============================================================

var fileContentMap = map[string]string{
	"configx/configx.go":       configxContent,
	"cronjob/cronjob.go":       cronjobContent,
	"middleware/middleware.go": middlewareContent,
	"model/model.go":           modelContent,
	"service/service.go":       serviceContent,
	"module/module.go":         moduleContent,
	"router/router.go":         routerContent,
	"dao/.gitkeep":             "",
	"provider/.gitkeep":        "",
}

// ============================================================
// 彩色输出工具
// ============================================================

var (
	green  = color.New(color.FgHiGreen).SprintFunc()
	yellow = color.New(color.FgHiYellow).SprintFunc()
	red    = color.New(color.FgHiRed).SprintFunc()
	cyan   = color.New(color.FgHiCyan).SprintFunc()
	gray   = color.New(color.FgHiBlack).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

func logSection(title string) {
	fmt.Printf("\n%s %s\n", cyan("▶"), bold(title))
}

func logSuccess(msg string) {
	fmt.Printf("  %s %s\n", green("✔"), msg)
}

func logInfo(msg string) {
	fmt.Printf("  %s %s\n", yellow("ℹ"), msg)
}

func logError(msg string) {
	fmt.Printf("  %s %s\n", red("✘"), msg)
}

func logFileCreate(filename string) {
	fmt.Printf("  %s %s\n", green("✔"), filename)
}

// ============================================================
// Run: 初始化新项目
// ============================================================

func Run(projectName string) error {
	projectDir := filepath.Base(projectName)

	// 项目目录
	logSection("Create Project Directory")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		logError("failed to create project directory")
		return err
	}
	logSuccess(projectDir)

	// 切换目录
	if err := os.Chdir(projectDir); err != nil {
		return err
	}

	// 初始化 Go module
	logSection("Initialize Go Module")
	logInfo(fmt.Sprintf("go mod init %s", projectName))
	cmd := exec.Command("go", "mod", "init", projectName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logError("go mod init failed")
		return err
	}
	logSuccess("Go module initialized")

	// 生成项目文件
	logSection("Generate Project Files")
	for file, content := range fileContentMap {
		if err := createFile(file, content); err != nil {
			logError(fmt.Sprintf("Failed to create %s", file))
			return err
		}
		logFileCreate(file)
	}

	// main.go
	if err := createFile("main.go", fmt.Sprintf(mainContent,
		projectName, projectName, projectName, projectName, projectName, projectName, projectName)); err != nil {
		return err
	}
	logFileCreate("main.go")

	// .gitignore
	if err := createFile(".gitignore", gitignoreContent); err != nil {
		return err
	}
	logFileCreate(".gitignore")

	// config.ini.example
	if err := createTeplConfig(); err != nil {
		return err
	}
	logFileCreate("config.ini.example")

	// 运行 go mod tidy
	logSection("Run Go Mod Tidy")
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		logError("go mod tidy failed")
		return err
	}
	logSuccess("Dependencies tidied")

	// 初始化 git 仓库
	logSection("Initialize Git Repository")
	cmd = exec.Command("git", "init")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		logError("git init failed")
		return err
	}
	logSuccess("Git repository initialized")

	// 最终提示
	logSection("Project Initialization Completed")
	fmt.Printf("\n%s Project %s created successfully!\n", green("🎉"), bold(projectDir))
	fmt.Println("\nNext steps:")
	fmt.Printf("  %s %s\n", cyan("$"), "cd "+projectDir)
	fmt.Printf("  %s %s\n", cyan("$"), "git add .")
	fmt.Printf("  %s %s\n", cyan("$"), "git commit -m \"Initial commit\"")

	return nil
}

// ============================================================
// 辅助函数
// ============================================================

func EnsureFileExists() error {
	for file, content := range fileContentMap {
		if _, err := os.Stat(file); err != nil && errors.Is(err, os.ErrNotExist) {
			if err := createFile(file, content); err != nil {
				return err
			}
		}
	}
	return nil
}

func createFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func createTeplConfig() error {
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	null, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	os.Stdout = null

	if err = config.Init(); err != nil {
		return err
	}
	defer config.Clean()

	// Create config file
	configFile, err := os.Create("config.ini")
	if err != nil {
		return err
	}
	defer configFile.Close()

	if err := config.Save(configFile); err != nil {
		return err
	}
	if err := os.Rename("config.ini", "config.ini.example"); err != nil {
		return err
	}
	return nil
}
