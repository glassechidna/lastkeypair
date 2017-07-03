package cmd

import (
	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/common"
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
		profile, _ := cmd.PersistentFlags().GetString("profile")
		region, _ := cmd.PersistentFlags().GetString("region")
		sess := common.AwsSession(profile, region)

		lambdaFunc, _ := cmd.PersistentFlags().GetString("lambda-func")
		kmsKeyId, _ := cmd.PersistentFlags().GetString("kms-key")
		funcIdentity, _ := cmd.PersistentFlags().GetString("func-identity")
		username, _ := cmd.PersistentFlags().GetString("ssh-username")

		common.SshExec(sess, lambdaFunc, funcIdentity, kmsKeyId, username, args)
	},
}

func init() {
	sshCmd.AddCommand(sshExecCmd)

	sshExecCmd.PersistentFlags().String("lambda-func", "LastKeypair", "Function name or ARN")
	sshExecCmd.PersistentFlags().String("kms-key", "alias/LastKeypair", "ID, ARN or alias of KMS key for auth to CA")
	sshExecCmd.PersistentFlags().String("func-identity", "LastKeypair", "")
	sshExecCmd.PersistentFlags().String("ssh-username", "ec2-user", "Username that you wish to SSH in with")
}
