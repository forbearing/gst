package main

import (
	"fmt"
	"os"

	"github.com/forbearing/gst/internal/codegen"
	"github.com/forbearing/gst/internal/codegen/gen"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "clean unused service files",
	Long:  "Clean unused service files that are no longer needed based on current model definitions",
	Run: func(cmd *cobra.Command, args []string) {
		pruneRun()
	},
}

func pruneRun() {
	if len(module) == 0 {
		var err error
		module, err = gen.GetModulePath()
		checkErr(err)
	}

	if !fileExists(modelDir) {
		logError(fmt.Sprintf("model dir not found: %s", modelDir))
		os.Exit(1)
	}

	// Scan all models
	logSection("Scan Models")
	allModels, err := codegen.FindModels(module, modelDir, serviceDir, excludes)
	checkErr(err)
	if len(allModels) == 0 {
		fmt.Println(gray("  No models found, pruning service files only"))
	} else {
		fmt.Printf("  %s %d models found\n", green("✔"), len(allModels))
	}

	// Scan existing service files
	oldServiceFiles := scanExistingServiceFiles(serviceDir)

	// Prune disabled service files
	logSection("Prune Disabled Service Files")
	if len(oldServiceFiles) > 0 {
		pruneServiceFiles(oldServiceFiles, allModels)
	} else {
		fmt.Printf("  %s No service files found to prune\n", green("✔"))
	}

	fmt.Printf("\n%s Code pruning completed successfully!\n", green("🎉"))
}
