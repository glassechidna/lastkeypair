package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"strings"
	"github.com/glassechidna/lastkeypair/common"
)

var sshMatchCmd = &cobra.Command{
	Use:   "match",
	Short: "Internal command invoked by SSH client",
	Long: "`ssh` invokes this to determine if LKP should be used to login to a host",
	Run: func(cmd *cobra.Command, args []string) {
		rei := common.NewReifiedLoginWithCmd(cmd, args)

		if !isLkpHost(rei.InstanceArn) {
			os.Exit(1)
		} else {
			rei.PopulateByInvoke()
		}
	},
}

func isLkpHost(hostname string) bool {
	return strings.HasPrefix(hostname, "arn:aws:ec2")
}

func init() {
	sshCmd.AddCommand(sshMatchCmd)
	sshMatchCmd.PersistentFlags().String("instance-arn", "", "")
	sshMatchCmd.PersistentFlags().String("ssh-username", "ec2-user", "")
}
