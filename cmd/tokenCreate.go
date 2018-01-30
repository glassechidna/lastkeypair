package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/glassechidna/lastkeypair/common"
	"log"
	"fmt"
	"encoding/json"
)

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		profile := viper.GetString("profile")
		region := viper.GetString("region")
		sess := common.ClientAwsSession(profile, region)

		key := viper.GetString("kms-key")
		fromName := viper.GetString("from-name")
		fromId := viper.GetString("from-id")
		fromAcct := viper.GetString("from-account")
		to := viper.GetString("to")
		typ := viper.GetString("principal")

		params := common.TokenParams{
			FromId: fromId,
			FromName: fromName,
			FromAccount: fromAcct,
			To: to,
			Type: typ,
		}

		ident, err := common.CallerIdentityUser(sess)
		if err != nil {
			log.Panicf("No 'from' specified and could not determine caller identity: %s", err.Error())
		}

		if len(params.FromName) == 0 {
			params.FromName = ident.Username
		}
		if len(params.FromAccount) == 0 {
			params.FromAccount = ident.AccountId
		}
		if len(params.FromId) == 0 {
			params.FromId = ident.UserId
		}

		token := common.CreateToken(sess, params, key)
		jsonToken, _ := json.Marshal(token)
		fmt.Println(string(jsonToken))
	},
}

func init() {
	tokenCmd.AddCommand(tokenCreateCmd)

	tokenCreateCmd.PersistentFlags().String("profile", "", "")
	tokenCreateCmd.PersistentFlags().String("region", "", "")

	tokenCreateCmd.PersistentFlags().String("kms-key", "alias/LastKeypair", "ID, ARN or alias of KMS key for auth to CA")
	tokenCreateCmd.PersistentFlags().String("from-account", "", "AWS account of 'from' user")
	tokenCreateCmd.PersistentFlags().String("from-name", "", "(defaults to IAM username)")
	tokenCreateCmd.PersistentFlags().String("from-id", "", "(defaults to IAM userid)")
	tokenCreateCmd.PersistentFlags().String("to", "", "")
	tokenCreateCmd.PersistentFlags().String("principal", "user", "")

	viper.BindPFlags(tokenCreateCmd.PersistentFlags())
}
