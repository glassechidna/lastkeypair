package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair"
	"fmt"
	"encoding/base64"
)

var tokenValidateCmd = &cobra.Command{
	Use:   "token-validate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		profile := viper.GetString("profile")
		region := viper.GetString("region")
		sess := lastkeypair.ClientAwsSession(profile, region)

		key := viper.GetString("key-id")
		fromName := viper.GetString("from-name")
		fromId := viper.GetString("from-id")
		fromAcct := viper.GetString("from-account")
		to := viper.GetString("to")
		typ := viper.GetString("type")
		signature := viper.GetString("signature")

		rawSig, _ := base64.StdEncoding.DecodeString(signature)

		token := lastkeypair.Token{
			Params: lastkeypair.TokenParams{
				FromId: fromId,
				FromName: fromName,
				FromAccount: fromAcct,
				To: to,
				Type: typ,
			},
			Signature: rawSig,
		}

		valid := lastkeypair.ValidateToken(sess, token, key)
		fmt.Printf("token valid: %+v\n", valid)
	},
}

func init() {
	advCmd.AddCommand(tokenValidateCmd)

	// profile is at root level
	tokenValidateCmd.PersistentFlags().String("region", "", "")

	tokenValidateCmd.PersistentFlags().String("key-id", "", "")
	tokenValidateCmd.PersistentFlags().String("from-id", "", "")
	tokenValidateCmd.PersistentFlags().String("from-name", "", "")
	tokenValidateCmd.PersistentFlags().String("from-account", "", "")
	tokenValidateCmd.PersistentFlags().String("to", "", "")
	tokenValidateCmd.PersistentFlags().String("type", "user", "")
	tokenValidateCmd.PersistentFlags().String("signature", "", "")

	//viper.BindPFlags(tokenValidateCmd.PersistentFlags())
}
