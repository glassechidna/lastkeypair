package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"io/ioutil"
	"github.com/glassechidna/lastkeypair/common"
	"log"
	"golang.org/x/crypto/ssh"
	"time"
)

var sshSignCmd = &cobra.Command{
	Use:   "sign",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		caKeyPath, _ := cmd.PersistentFlags().GetString("ca-key-path")
		userKeyPath, _ := cmd.PersistentFlags().GetString("user-key-path")
		keyId, _ := cmd.PersistentFlags().GetString("key-id")
		duration, _ := cmd.PersistentFlags().GetInt64("duration")
		principals, _ := cmd.PersistentFlags().GetStringSlice("principals")

		keyBytes, _ := ioutil.ReadFile(caKeyPath)
		userPubkeyBytes, _ := ioutil.ReadFile(userKeyPath)

		formatted, err := common.SignSsh(
			keyBytes,
			userPubkeyBytes,
			ssh.UserCert,
			uint64(time.Now().Unix() + duration),
			common.DefaultSshPermissions,
			keyId,
			principals,
		)

		if err != nil {
			log.Panicf("err signing ssh key: %s", err.Error())
		}

		fmt.Println(*formatted)
	},
}

func init() {
	sshCmd.AddCommand(sshSignCmd)

	sshSignCmd.PersistentFlags().String("ca-key-path", "", "")
	sshSignCmd.PersistentFlags().String("user-key-path", "", "")
	sshSignCmd.PersistentFlags().String("key-id", "", "")
	sshSignCmd.PersistentFlags().Int64("duration", 3600, "")
	sshSignCmd.PersistentFlags().StringSlice("principals", []string{}, "")
}
