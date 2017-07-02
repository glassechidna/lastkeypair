package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/common"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output lastkeypair version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\nBuild Date: %s\n", common.ApplicationVersion, common.ApplicationBuildDate)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
