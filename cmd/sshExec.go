package cmd

import (
	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair"
	"os/exec"
	"syscall"
	"os"
	"fmt"
	"strings"
	"github.com/spf13/viper"
)

var sshExecCmd = &cobra.Command{
	Use:   "exec",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		rei := lastkeypair.NewReifiedLoginWithCmd(cmd, args)
		rei.PopulateByInvoke()

		sshconfPath := rei.WriteSshConfig()
		sshcmd := []string{"ssh", "-F", sshconfPath}
		sshcmd = append(sshcmd, args...)
		if len(rei.Response.TargetAddress) > 0 {
			sshcmd = append(sshcmd, "target")
		}

		dryRun, _ := cmd.PersistentFlags().GetBool("dry-run")
		if dryRun {
			fmt.Println(strings.Join(sshcmd, " "))
		} else {
			sshPath, _ := exec.LookPath("ssh")
			syscall.Exec(sshPath, sshcmd, os.Environ())
		}
	},
}

func init() {
	sshCmd.AddCommand(sshExecCmd)

	sshExecCmd.PersistentFlags().String("lambda-func", "LastKeypair", "Function name or ARN")
	sshExecCmd.PersistentFlags().String("kms-key", "alias/LastKeypair", "ID, ARN or alias of KMS key for auth to CA")
	sshExecCmd.PersistentFlags().String("instance-arn", "", "")
	sshExecCmd.PersistentFlags().String("ssh-username", "ec2-user", "Username that you wish to SSH in with")
	sshExecCmd.PersistentFlags().StringSlice("voucher", []string{}, "Optional voucher(s) from other people")
	sshExecCmd.PersistentFlags().Bool("dry-run", false, "Do everything _except_ the SSH login")

	viper.BindPFlags(sshExecCmd.PersistentFlags())
}
