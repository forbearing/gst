package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

func checkErr(err error) {
	if err == nil {
		return
	}
	panic(err)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func ensureParentDir(filename string) error {
	dir := filepath.Dir(filename)

	var err error
	if _, err = os.Stat(dir); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}
	return err
}

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

func logCreate(filename string) {
	fmt.Printf("  %s %s\n", green("✔ CREATE"), filename)
}

func logUpdate(filename string) {
	fmt.Printf("  %s %s\n", yellow("✔ UPDATE"), filename)
}

func logSkip(filename string) {
	fmt.Printf("  %s %s\n", gray("→ SKIP"), filename)
}

func logError(msg string) {
	fmt.Printf("  %s %s\n", red("✘ ERROR"), msg)
}

func writeFileWithLog(filename string, content string) {
	if fileExists(filename) {
		oldData, err := os.ReadFile(filename)
		checkErr(err)
		if string(oldData) == content {
			logSkip(filename)
		} else {
			logUpdate(filename)
			checkErr(os.WriteFile(filename, []byte(content), 0o600))
		}
	} else {
		logCreate(filename)
		checkErr(ensureParentDir(filename))
		checkErr(os.WriteFile(filename, []byte(content), 0o600))
	}
}
