package main

import (
	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Integration with the client ssh program",
}

func init() {
	RootCmd.AddCommand(sshCmd)
}
