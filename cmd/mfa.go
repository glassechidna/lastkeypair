package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/glassechidna/awscredcache"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"time"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pquerna/otp/totp"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
)

var mfaCmd = &cobra.Command{
	Use:   "mfa",
	Short: "MFA-authenticate your AWS credentials before SSHing",
	Long: `
Some AWS account administrators may require users to authenticate using MFA
in order to SSH into an instance. SSH doesn't support interactive helper tools,
so you have to type in your code using this command before you can SSH like normal.
`,
	Run: func(cmd *cobra.Command, args []string) {
		code, _ := cmd.Flags().GetString("code")
		duration, _ := cmd.Flags().GetInt("duration")
		profile := viper.GetString("profile")
		mfa(profile, code, duration)
	},
}

func mfa(profile, code string, duration int) {
	provider := awscredcache.NewAwsCacheCredProvider(profile)
	provider.Duration = time.Duration(duration) * time.Second

	provider.MfaCodeProvider = func(mfaSecret string) (string, error) {
		if len(mfaSecret) > 0 {
			return totp.GenerateCode(mfaSecret, time.Now())
		} else if len(code) > 0 {
			return code, nil
		} else {
			return stscreds.StdinTokenProvider()
		}
	}

	creds := credentials.NewCredentials(provider)

	sess, err := session.NewSession(&aws.Config{Credentials: creds})
	if err != nil { panic(err) }

	api := sts.New(sess)
	resp, err := api.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil { panic(err) }

	fmt.Printf("Successfully authenticated as %s\n", *resp.Arn)
}

func init() {
	RootCmd.AddCommand(mfaCmd)
	mfaCmd.Flags().StringP("code", "c", "", "6 digit MFA code")
	mfaCmd.Flags().IntP("duration", "d", 43200, "Validity of cached credentials (in seconds)")
	viper.BindPFlags(mfaCmd.PersistentFlags())
}
