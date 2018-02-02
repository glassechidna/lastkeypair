package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
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
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
