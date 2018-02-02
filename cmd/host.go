package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"io/ioutil"
	"github.com/pkg/errors"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"log"
	"github.com/glassechidna/lastkeypair/common"
	"github.com/aws/aws-sdk-go/aws"
	"os"
	"strings"
	"path/filepath"
)

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Create signed SSH host certificates",
	Long: `
A signed SSH host certificate means that users are able to log into a machine
without seeing a host key validation prompt if their SSH client trusts the 
certificate authority that signed the host cert.

This command can be invoked from an EC2 instance userdata script to request
a signed SSH host cert and install it in the appropriate sshd config.
`,
	Run: func(cmd *cobra.Command, args []string) {
		hostKeyPath, _ := cmd.PersistentFlags().GetString("host-key-path")
		signedHostKeyPath, _ := cmd.PersistentFlags().GetString("signed-host-key-path")
		caPubkeyPath, _ := cmd.PersistentFlags().GetString("cert-authority-path")
		sshdConfigPath, _ := cmd.PersistentFlags().GetString("sshd-config-path")
		authorizedPrincipalsPath, _ := cmd.PersistentFlags().GetString("authorized-principals-path")
		functionName, _ := cmd.PersistentFlags().GetString("lambda-name")
		kmsKeyId, _ := cmd.PersistentFlags().GetString("kms-key")
		principals, _ := cmd.PersistentFlags().GetStringSlice("principal")

		err := doit(hostKeyPath, signedHostKeyPath, caPubkeyPath, sshdConfigPath, authorizedPrincipalsPath, functionName, kmsKeyId, principals)
		if err != nil {
			log.Panicf("err: %s\n", err.Error())
		}
	},
}

func hostSession() (*session.Session, error) {
	sessOpts := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating aws session")
	}

	client := ec2metadata.New(sess)
	if client.Available() {
		region, err := client.Region()
		if err != nil {
			return nil, errors.Wrap(err, "getting region from ec2 metadata")
		}
		sess = sess.Copy(aws.NewConfig().WithRegion(region))
	}

	return sess, nil
}

func doit(hostKeyPath, signedHostKeyPath, caPubkeyPath, sshdConfigPath, authorizedPrincipalsPath, functionName, kmsKeyId string, principals []string) error {
	// we absolute-ize these paths because ssh requires paths in sshd_config to be absolute
	authorizedPrincipalsPath, _ = filepath.Abs(authorizedPrincipalsPath)
	caPubkeyPath, _ = filepath.Abs(caPubkeyPath)
	signedHostKeyPath, _ = filepath.Abs(signedHostKeyPath)


	hostKeyBytes, err := ioutil.ReadFile(hostKeyPath)
	if err != nil {
		return errors.Wrap(err, "reading ssh host key")
	}
	hostKey := string(hostKeyBytes)

	sess, err := hostSession()
	client := ec2metadata.New(sess)

	ident, err := common.CallerIdentityUser(sess)
	instanceArn, err := getInstanceArn(client)
	if err != nil {
		return errors.Wrap(err, "fetching instance arn from metadata service")
	}

	principals = append(principals, *instanceArn)
	token, err := hostCertToken(sess, *ident, kmsKeyId, *instanceArn, principals)

	caPubkey, err := client.GetMetadata("public-keys/0/openssh-key")
	if err != nil {
		return errors.Wrap(err, "fetching ssh CA key")
	}

	response := common.HostCertRespJson{}
	err = common.RequestSignedPayload(sess, functionName, common.HostCertReqJson{
		EventType: "HostCertReq",
		Token: *token,
		PublicKey: hostKey,
	}, &response)
	if err != nil {
		return errors.Wrap(err, "requesting signed host key")
	}

	err = ioutil.WriteFile(signedHostKeyPath, []byte(response.SignedHostPublicKey), 0600)
	if err != nil {
		return errors.Wrap(err, "writing signed host key to filesystem")
	}

	err = ioutil.WriteFile(caPubkeyPath, []byte(caPubkey), 0600)
	if err != nil {
		return errors.Wrap(err, "writing ca pubkey to filesystem")
	}

	authorizedPrincipalsBytes := []byte(fmt.Sprintf("%s\n", strings.Join(principals, "\n")))

	err = ioutil.WriteFile(authorizedPrincipalsPath, authorizedPrincipalsBytes, 0444)
	if err != nil {
		return errors.Wrap(err, "writing authorized principals to filesystem")
	}

	err = appendToFile(sshdConfigPath, fmt.Sprintf(`
HostCertificate %s
TrustedUserCAKeys %s
AuthorizedPrincipalsFile %s
`, signedHostKeyPath, caPubkeyPath, authorizedPrincipalsPath))
	if err != nil {
		return errors.Wrap(err, "appending to sshd config")
	}

	return nil
}

func getInstanceArn(client *ec2metadata.EC2Metadata) (*string, error) {
	region, err := client.Region()
	if err != nil {
		return nil, errors.Wrap(err, "getting region")
	}

	ident, err := client.GetInstanceIdentityDocument()
	if err != nil {
		return nil, errors.Wrap(err, "getting identity doc for account id and instance id")
	}

	ret := fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, ident.AccountID, ident.InstanceID)
	return &ret, nil
}

func hostCertToken(sess *session.Session, ident common.StsIdentity, kmsKeyId, instanceArn string, principals []string) (*common.Token, error) {
	params := common.TokenParams{
		FromId:          ident.UserId,
		FromAccount:     ident.AccountId,
		To:              "LastKeypair",
		Type:            "AssumedRole",
		HostInstanceArn: instanceArn,
		Principals: principals,
	}

	ret := common.CreateToken(sess, params, kmsKeyId)
	return &ret, nil
}

func appendToFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	RootCmd.AddCommand(hostCmd)

	hostCmd.PersistentFlags().String("host-key-path", "/etc/ssh/ssh_host_rsa_key.pub", "")
	hostCmd.PersistentFlags().String("signed-host-key-path", "/etc/ssh/ssh_host_rsa_key-cert.pub", "")
	hostCmd.PersistentFlags().String("cert-authority-path", "/etc/ssh/cert_authority.pub", "")
	hostCmd.PersistentFlags().String("authorized-principals-path", "/etc/ssh/authorized_principals", "")
	hostCmd.PersistentFlags().String("sshd-config-path", "/etc/ssh/sshd_config", "")
	hostCmd.PersistentFlags().String("lambda-name", "LastKeypair", "")
	hostCmd.PersistentFlags().StringSlice("principal", []string{""}, "Additional principals to request from CA")
	hostCmd.PersistentFlags().String("kms-key", "alias/LastKeypair", "ID, ARN or alias of KMS key for auth to CA")
}
