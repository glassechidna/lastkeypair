package cmd

import (
	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/common"
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
		profile := viper.GetString("profile")
		region, _ := cmd.PersistentFlags().GetString("region")
		sess := common.ClientAwsSession(profile, region)

		lambdaFunc := viper.GetString("lambda-func")
		kmsKeyId := viper.GetString("kms-key")
		instanceArn, _ := cmd.PersistentFlags().GetString("instance-arn")
		username, _ := cmd.PersistentFlags().GetString("ssh-username")
		dryRun, _ := cmd.PersistentFlags().GetBool("dry-run")
		vouchers, _ := cmd.PersistentFlags().GetStringSlice("voucher")

		sshcmd := common.SshCommand(sess, lambdaFunc, kmsKeyId, instanceArn, username, vouchers, args)

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
