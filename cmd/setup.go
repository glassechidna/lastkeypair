package cmd

import (
	"io/ioutil"
	"path"
	"fmt"
	"os"
	"strings"
	"sort"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/go-ini/ini"
	"github.com/mitchellh/go-homedir"
	"github.com/inconshreveable/mousetrap"
	"github.com/glassechidna/lastkeypair/common"
	"github.com/glassechidna/awscredcache/sneakyvendor/aws-shared-defaults"
	"github.com/AlecAivazis/survey"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "First-time installation and setup",
	Run: func(cmd *cobra.Command, args []string) {
		setup()
	},
}

func setup() {
	profile := selectAwsProfile()
	lambda := inputLambdaFunc()
	kms := inputKmsKey()
	writeLkpConfig(profile, lambda, kms)
	askUserAboutMfa(profile)
	writeSshConfig()
	addIncludeToSshConfig("~/.lkp/config") // openssh on windows doesn't like a non-relative path
	promptToAddToPath()
	informNextSteps()
}

func askUserAboutMfa(profile string) {
	prompt := &survey.Confirm{
		Message: "Does your administrator require MFA to use LKP?",
	}

	mfaRequired := false
	survey.AskOne(prompt, &mfaRequired, nil)

	if mfaRequired {
		cfgPath := shareddefaults.SharedConfigFilename()
		cfg, _ := ini.Load(cfgPath)
		sect, err := cfg.GetSection(profile)
		if err != nil {
			sect, _ = cfg.GetSection("profile " + profile)
		}

		_, err = sect.GetKey("mfa_serial")
		if err == nil {
			fmt.Printf(`
Before SSHing into instances, you should first execute 'lastkeypair mfa' 
and follow the prompts.
`)
		} else {
			fmt.Printf(`
You should add an mfa_serial = arn:aws:iam::XXXXXX:mfa/SERIAL line to your
~/.aws/config in the [%s] section in order to make MFA work. Before SSHing
into instances, you should first execute 'lastkeypair mfa' and follow the
prompts.
`, sect.Name())
		}
	}
}

func selectAwsProfile() string {
	profiles := awsProfileNames()

	prompt := &survey.Select{
		Message: "Which AWS profile do you want to use with LKP by default?",
		Options: profiles,
		PageSize: 15,
	}

	result := ""
	err := survey.AskOne(prompt, &result, nil)
	if err != nil {
		panic(err)
	}

	return result
}

func inputLambdaFunc() string {
	prompt := &survey.Input{
		Message: "What is the name/ARN of the LKP Lambda function?",
		Default: "LastKeypair",
	}

	result := ""
	err := survey.AskOne(prompt, &result, nil)
	if err != nil {
		panic(err)
	}

	return result
}

func inputKmsKey() string {
	prompt := &survey.Input{
		Message: "What is the alias/key ID/ARN of the KMS key ID?",
		Default: "alias/LastKeypair",
	}

	result := ""
	err := survey.AskOne(prompt, &result, nil)
	if err != nil {
		panic(err)
	}

	return result
}

func writeLkpConfig(profile, lambda, kms string) {
	str := fmt.Sprintf(`
profile: %s
lambda-func: %s
kms-key: %s
`, profile, lambda, kms)

	ioutil.WriteFile(path.Join(common.AppDir(), "config.yml"), []byte(str), 0644)
}

func writeSshConfig() string {
	str := `
Match exec "lastkeypair ssh match --instance-arn %n --ssh-username %r"
  IdentityFile /Users/aidan/.lkp/id_rsa
  CertificateFile /Users/aidan/.lkp/id_rsa-cert.pub
  ProxyCommand lastkeypair ssh proxy --instance-arn %h
`

	lkpSshConfigPath := path.Join(common.AppDir(), "ssh_config")
	ioutil.WriteFile(lkpSshConfigPath, []byte(str), 0644)
	return lkpSshConfigPath
}

func addIncludeToSshConfig(path string) {
	sshConfigPath, _ := homedir.Expand("~/.ssh/config")
	sshConfigBytes, _ := ioutil.ReadFile(sshConfigPath)

	sshConfig := string(sshConfigBytes)
	sshConfig = fmt.Sprintf("Include %s\n\n%s", path, sshConfig)

	os.MkdirAll(filepath.Dir(sshConfigPath), 0644)
	ioutil.WriteFile(sshConfigPath, []byte(sshConfig), 0644)
}

func promptToAddToPath() {
	pathenv := os.Getenv("PATH")
	paths := filepath.SplitList(pathenv)
	sort.Strings(paths)
	joined := strings.Join(paths, "\n")

	fmt.Printf(`
The last step is to now add LastKeypair to somewhere on your PATH. Consider
one of the following directories already on your PATH:

%s
`, joined)

}

func keepTerminalVisible() {
	// terminal on macos stays open after exe ends. TODO check linux behaviour
	if mousetrap.StartedByExplorer() {
		var input string
		fmt.Scanln(&input)
	}
}

func informNextSteps() {
	fmt.Println(`
Great work, you should be all set up now. Configuration files have been written
to ~/.ssh/config and ~/.lkp/config.yml. You can now run 'ssh ec2-user@instance-arn'
and hit the ground running.`)
}

func init() {
	RootCmd.AddCommand(setupCmd)
}

func awsProfileNames() []string {
	cfgPath := shareddefaults.SharedConfigFilename()
	cfg, err := ini.Load(cfgPath)

	if err != nil {
		fmt.Fprintf(os.Stderr, `
LastKeypair requires that you have a valid configuration file stored at %s.
This file will look something like:

	[default]
	region = ap-southeast-2
	mfa_serial = arn:aws:iam::0987654321:mfa/aidan.steele@example.com

You will also need a corresponding credentials file stored at %s with 
contents that look like:

	[default]
	aws_access_key_id = AKIA...
	aws_secret_access_key = qGrg....

LastKeypair will also work with named profiles if they are defined in your
configuration file.

Hit Enter now to close this prompt. After the above files are created, you
can open LastKeypair again.
`, cfgPath, shareddefaults.SharedCredentialsFilename())
	}

	rawProfiles := cfg.SectionStrings()
	profiles := []string{}

	for _, profile := range rawProfiles {
		if strings.HasPrefix(profile, "profile ") {
			profiles = append(profiles, profile[8:])
		} else if profile != "DEFAULT" {
			profiles = append(profiles, profile)
		}
	}

	sort.Strings(profiles)
	return profiles
}
