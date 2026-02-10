package main

import "github.com/spf13/cobra"

var (
	modelDir   string = "model"
	serviceDir string = "service"
	routerDir  string = "router"
	daoDir     string = "dao"
	excludes   []string
	module     string
	debug      bool
	prune      bool
)

var rootCmd = &cobra.Command{
	Use:     "gg",
	Short:   "gst code generator",
	Long:    "gst code generator",
	Version: "1.0.0",
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&prune, "prune", false, "Prune disabled service action files with user confirmation")

	rootCmd.AddCommand(genCmd,
		watchCmd,
		newCmd,
		astCmd,
		pruneCmd,
		checkCmd,
		routesCmd,
		dockerCmd,
		k8sCmd,
		buildCmd,
		releaseCmd,
		configCmd,
	)
}
