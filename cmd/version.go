package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var ApplicationVersion string
var ApplicationBuildDate string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output lastkeypair version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\nBuild Date: %s\n", ApplicationVersion, ApplicationBuildDate)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
