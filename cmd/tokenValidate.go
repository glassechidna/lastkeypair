// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/glassechidna/lastkeypair/common"
	"fmt"
	"encoding/base64"
)

// tokenValidateCmd represents the tokenValidate command
var tokenValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Work your own magic here
		profile := viper.GetString("profile")
		region := viper.GetString("region")
		sess := common.AwsSession(profile, region)

		key := viper.GetString("key-id")
		from := viper.GetString("from")
		fromAcct := viper.GetString("from-account")
		to := viper.GetString("to")
		typ := viper.GetString("type")
		signature := viper.GetString("signature")

		rawSig, _ := base64.StdEncoding.DecodeString(signature)

		token := common.Token{
			Params: common.TokenParams{
				KeyId: key,
				From: from,
				FromAccount: fromAcct,
				To: to,
				Type: typ,
			},
			Signature: rawSig,
		}


		valid := common.ValidateToken(sess, token)
		fmt.Printf("token valid: %+v\n", valid)
	},
}

func init() {
	tokenCmd.AddCommand(tokenValidateCmd)

	tokenValidateCmd.PersistentFlags().String("profile", "", "")
	tokenValidateCmd.PersistentFlags().String("region", "", "")

	tokenValidateCmd.PersistentFlags().String("key-id", "", "")
	tokenValidateCmd.PersistentFlags().String("from", "", "")
	tokenValidateCmd.PersistentFlags().String("from-account", "", "")
	tokenValidateCmd.PersistentFlags().String("to", "", "")
	tokenValidateCmd.PersistentFlags().String("type", "user", "")
	tokenValidateCmd.PersistentFlags().String("signature", "", "")

	//viper.BindPFlags(tokenValidateCmd.PersistentFlags())
}
