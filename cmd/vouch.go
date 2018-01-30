package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/common"
	"github.com/spf13/viper"
)

var vouchCmd = &cobra.Command{
	Use:   "vouch",
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

		keyId, _ := cmd.PersistentFlags().GetString("kms-key")
		to, _ := cmd.PersistentFlags().GetString("to")
		vouchee, _ := cmd.PersistentFlags().GetString("vouchee")
		context, _ := cmd.PersistentFlags().GetString("context")

		token := common.Vouch(sess, keyId, to, vouchee, context)
		encoded := token.Encode()
		fmt.Println(encoded)
	},
}

func init() {
	RootCmd.AddCommand(vouchCmd)

	// profile is at root level
	vouchCmd.PersistentFlags().String("region", "", "")

	vouchCmd.PersistentFlags().String("kms-key", "alias/LastKeypair", "ID, ARN or alias of KMS key for auth to CA")
	vouchCmd.PersistentFlags().String("to", "LastKeypair", "")

	vouchCmd.PersistentFlags().String("vouchee", "", "")
	vouchCmd.PersistentFlags().String("context", "", "")
}
