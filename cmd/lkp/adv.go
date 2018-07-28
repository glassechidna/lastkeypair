package main

import (
	"github.com/spf13/cobra"
)

var advCmd = &cobra.Command{
	Use:   "adv",
	Short: "A grab bag of commands useful to the lkp developer :)",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	RootCmd.AddCommand(advCmd)
}
