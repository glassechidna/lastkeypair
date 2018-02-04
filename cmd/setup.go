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
	"github.com/manifoldco/promptui"
	"github.com/mitchellh/go-homedir"
	"github.com/inconshreveable/mousetrap"
	"github.com/glassechidna/lastkeypair/common"
	"github.com/glassechidna/awscredcache/sneakyvendor/aws-shared-defaults"
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
	os.Exit(1)

	lkpSsh := writeSshConfig()
	addIncludeToSshConfig(lkpSsh)
	promptToAddToPath()
	informNextSteps()
	keepTerminalVisible()
}

func askUserAboutMfa(profile string) {
	prompt := promptui.Select{
		Label: "Does your administrator require MFA to use LKP?",
		Items: []string{"Yes", "No"},
	}

	_, result, _ := prompt.Run()

	if result == "Yes" {
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

	prompt := promptui.Select{
		Label: "Which AWS profile do you want to use with LKP by default?",
		Items: profiles,
		Size: 15,
	}

	_, result, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	return result
}

func inputLambdaFunc() string {
	prompt := promptui.Prompt{
		Label: "What is the name/ARN of the LKP Lambda function?",
		Default: "LastKeypair",
	}

	result, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	return result
}

func inputKmsKey() string {
	prompt := promptui.Prompt{
		Label: "What is the alias/key ID/ARN of the KMS key ID?",
		Default: "alias/LastKeypair",
	}

	result, err := prompt.Run()
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
Match exec "lastkeypair ssh match --instance-arn %n"
  IdentityFile ~/.lkp/id_rsa
  CertificateFile ~/.lkp/id_rsa-cert.pub
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

	ioutil.WriteFile(sshConfigPath, []byte(sshConfig), 0644)
}

func promptToAddToPath() {
	pathenv := os.Getenv("PATH")
	paths := filepath.SplitList(pathenv)
	sort.Strings(paths)
	joined := strings.Join(paths, "\n")

	fmt.Sprintf(`
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
	cfg, _ := ini.Load(cfgPath)

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
