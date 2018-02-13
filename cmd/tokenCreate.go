package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/glassechidna/lastkeypair/common"
	"log"
	"fmt"
	"encoding/json"
	"strings"
)

var tokenCreateCmd = &cobra.Command{
	Use:   "token-create",
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

		pairs := viper.GetStringSlice("context")
		userContext := pairsToMap(pairs, "=")

		params := common.TokenParams{
			FromId: fromId,
			FromName: fromName,
			FromAccount: fromAcct,
			To: to,
			Type: typ,
			UserProvided: userContext,
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

func pairsToMap(pairs []string, delim string) map[string]string {
	outMap := map[string]string{}

	for _, pair := range pairs {
		splitted := strings.SplitN(pair, delim, 2)
		key := splitted[0]
		val := splitted[1]
		outMap[key] = val
	}

	return outMap
}

func init() {
	advCmd.AddCommand(tokenCreateCmd)

	// profile is at root level
	tokenCreateCmd.PersistentFlags().String("region", "", "")

	tokenCreateCmd.PersistentFlags().String("kms-key", "alias/LastKeypair", "ID, ARN or alias of KMS key for auth to CA")
	tokenCreateCmd.PersistentFlags().String("from-account", "", "AWS account of 'from' user")
	tokenCreateCmd.PersistentFlags().String("from-name", "", "(defaults to IAM username)")
	tokenCreateCmd.PersistentFlags().String("from-id", "", "(defaults to IAM userid)")
	tokenCreateCmd.PersistentFlags().String("to", "", "")
	tokenCreateCmd.PersistentFlags().String("principal", "user", "")
	tokenCreateCmd.PersistentFlags().StringSlice("context", []string{}, "additional key=val pairs to include in the context")

	viper.BindPFlags(tokenCreateCmd.PersistentFlags())
}
