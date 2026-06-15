package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forbearing/gst/internal/clioutput"
	"github.com/forbearing/gst/internal/codegen/constants"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "watch model files and auto regenerate code when changes detected",
	Long: `Watch mode monitors your model directory for changes and automatically
regenerates service, router and other code when modifications are detected.

This is useful during development to avoid manually running 'gg gen' after
every model change.

Example:
  gg watch                    # watch model directory with default settings
  gg watch --model-dir ./pkg  # watch custom model directory
  gg watch --debounce 500ms   # set custom debounce duration
`,
	Run: func(cmd *cobra.Command, args []string) {
		watchRun()
	},
}

var debounce time.Duration

func init() {
	watchCmd.Flags().DurationVar(&debounce, "debounce", 300*time.Millisecond, "debounce duration to avoid multiple regenerations")
}

func watchRun() {
	// Validate model directory exists
	if !fileExists(modelDir) {
		clioutput.Error("", "model directory not found: %s", modelDir)
		os.Exit(1)
	}

	// Create file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		clioutput.Error("", "failed to create watcher: %v", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Add model directory to watcher
	if err := addDirRecursive(watcher, modelDir); err != nil {
		clioutput.Error("", "failed to watch directory: %v", err)
		os.Exit(1)
	}

	clioutput.Section("Watch")
	clioutput.Success("", "Watch mode started")
	clioutput.Item("", "Watching directory: %s", modelDir)
	clioutput.Item("", "Debounce duration: %v", debounce)
	clioutput.Line(clioutput.StyleMuted, "Press Ctrl+C to stop")

	// Run initial generation
	clioutput.Info("", "Running initial code generation...")
	genRun()
	clioutput.Success("", "Initial generation completed")

	// Debounce timer to avoid multiple regenerations
	var timer *time.Timer
	var pendingRegeneration bool

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only watch .go files, skip test files
			if !strings.HasSuffix(event.Name, constants.ExtensionGo) ||
				strings.HasSuffix(event.Name, constants.PatternTestFile) {
				continue
			}

			// Ignore files in vendor, testdata directories
			if shouldIgnoreFile(event.Name) {
				continue
			}

			// Handle different event types
			switch {
			case event.Has(fsnotify.Write), event.Has(fsnotify.Create):
				fmt.Printf(
					"%s %s %s\n",
					clioutput.Text(clioutput.StyleMuted, "%s", time.Now().Format("15:04:05")),
					clioutput.Text(clioutput.StyleWarn, "File changed:"),
					filepath.Base(event.Name),
				)

				// Reset or create debounce timer
				if timer != nil {
					timer.Stop()
				}
				pendingRegeneration = true
				timer = time.AfterFunc(debounce, func() {
					if pendingRegeneration {
						fmt.Println()
						clioutput.Info("", "Regenerating code...")
						genRun()
						clioutput.Success("", "Regeneration completed")
						pendingRegeneration = false
					}
				})

			case event.Has(fsnotify.Remove):
				// If a directory is removed, we may need to remove it from watcher
				// But fsnotify will handle this automatically
				fmt.Printf(
					"%s %s %s\n",
					clioutput.Text(clioutput.StyleMuted, "%s", time.Now().Format("15:04:05")),
					clioutput.Text(clioutput.StyleError, "File removed:"),
					filepath.Base(event.Name),
				)

				// Trigger regeneration for removal too
				if timer != nil {
					timer.Stop()
				}
				pendingRegeneration = true
				timer = time.AfterFunc(debounce, func() {
					if pendingRegeneration {
						fmt.Println()
						clioutput.Info("", "Regenerating code...")
						genRun()
						clioutput.Success("", "Regeneration completed")
						pendingRegeneration = false
					}
				})

			case event.Has(fsnotify.Rename):
				fmt.Printf(
					"%s %s %s\n",
					clioutput.Text(clioutput.StyleMuted, "%s", time.Now().Format("15:04:05")),
					clioutput.Text(clioutput.StyleWarn, "File renamed:"),
					filepath.Base(event.Name),
				)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			clioutput.Error("", "watcher error: %v", err)
		}
	}
}

// addDirRecursive recursively adds directories to watcher
func addDirRecursive(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and testdata directories
		if info.IsDir() {
			base := filepath.Base(path)
			if base == constants.DirVendor || base == constants.DirTestData {
				return filepath.SkipDir
			}

			// Add directory to watcher
			if err := watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch %s: %w", path, err)
			}
		}

		return nil
	})
}

// shouldIgnoreFile checks if a file should be ignored by watcher
func shouldIgnoreFile(filename string) bool {
	// Ignore files in vendor or testdata
	if strings.Contains(filename, "/"+constants.DirVendor+"/") ||
		strings.Contains(filename, "/"+constants.DirTestData+"/") {
		return true
	}

	// Ignore files starting with underscore
	base := filepath.Base(filename)
	if strings.HasPrefix(base, constants.PrefixIgnore) {
		return true
	}

	// Ignore generated files (optional: you might want to watch these too)
	// if strings.Contains(filename, "/service/") ||
	// 	strings.Contains(filename, "/router/") {
	// 	return true
	// }

	return false
}
