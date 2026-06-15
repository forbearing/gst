package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/forbearing/gst/internal/clioutput"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [flags] [-- air-args...]",
	Short: "Run the application with hot reload using air",
	Long: `Run the application with hot reload using air.

This command uses air (https://github.com/cosmtrek/air) for hot reloading during development.
If air is not installed, it will be automatically installed.

Examples:
  gg run                    # Run with default air configuration
  gg run -- -c .air.toml    # Run with custom air configuration
  gg run -- --help          # Show air help`,
	RunE: runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	clioutput.Section("Check Air Installation")

	// Check if air is installed
	if !isAirInstalled() {
		clioutput.Item("", "Air is not installed, installing...")
		if err := installAir(); err != nil {
			clioutput.Error("", "Failed to install air: %v", err)
			return err
		}
		clioutput.Success("", "Air installed successfully")
	} else {
		clioutput.Success("", "Air is already installed")
	}

	clioutput.Section("Start Development Server")

	// Prepare air command with arguments
	airArgs := []string{}
	if len(args) > 0 {
		airArgs = args
	}

	// Run air command
	clioutput.Info("", "Starting air with hot reload...")
	airCmd := exec.Command("air", airArgs...)
	airCmd.Stdout = os.Stdout
	airCmd.Stderr = os.Stderr
	airCmd.Stdin = os.Stdin

	if err := airCmd.Run(); err != nil {
		clioutput.Error("", "Air command failed: %v", err)
		return err
	}

	return nil
}

// isAirInstalled checks if air is installed
func isAirInstalled() bool {
	_, err := exec.LookPath("air")
	return err == nil
}

// installAir installs air using go install
func installAir() error {
	cmd := exec.Command("go", "install", "github.com/air-verse/air@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install air: %w", err)
	}

	return nil
}
