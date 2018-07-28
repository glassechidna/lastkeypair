package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output lastkeypair version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\nBuild Date: %s\n", lastkeypair.ApplicationVersion, lastkeypair.ApplicationBuildDate)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
