package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/inconshreveable/mousetrap"
	"os"
	"github.com/mitchellh/go-homedir"
	"fmt"
)

var cfgFile string

var RootCmd = &cobra.Command{
	Use:   "lastkeypair",
	Short: "A serverless SSH certificate authority to control access to machines using IAM and Lambda",
	Long: `
lastkeypair is a CLI tool and a Lambda function to control remote access to 
instances through SSH certificates. User certificates are created on-demand and
are specific to a user and target server. Host certificates are also created
at instance launch time and allow for host validation without needing host key
validation prompts.
`,
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func Execute() {
	if mousetrap.StartedByExplorer() {
		configPath, _ := homedir.Expand("~/.lkp/config.yml")
		defer keepTerminalVisible()

		if !fileExists(configPath) {
			setup()
		} else {
			fmt.Println("LastKeypair is a command-line tool, you should invoke it from the command prompt or Powershell.")
		}
	} else {
		if err := RootCmd.Execute(); err != nil {

		}
	}
}

func init() {
	cobra.MousetrapHelpText = "" // we want to use mousetrap ourselves in setup
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lkp/config.yml)")
	RootCmd.PersistentFlags().StringP("profile", "p", "", "Name of profile in ~/.aws/config to use")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.lkp")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
	}
}
