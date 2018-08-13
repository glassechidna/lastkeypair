package lastkeypair

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"io/ioutil"
	"github.com/aws/aws-sdk-go/service/lambda"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"fmt"
	"strings"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"path/filepath"
	"os"
)

func sshReqResp(sess *session.Session, lambdaFunc, kmsKeyId, instanceArn, username string, encodedVouchers []string) (UserCertReqJson, UserCertRespJson) {
	kp, _ := MyKeyPair()

	ident, err := CallerIdentityUser(sess)
	if err != nil {
		log.Panicf("error getting aws user identity: %+v\n", err)
	}

	vouchers := []VoucherToken{}
	for _, encVoucher := range encodedVouchers {
		voucher, err := DecodeVoucherToken(encVoucher)
		if err != nil {
			log.Panicf("couldn't decode voucher: %+v\n", err)
		}
		vouchers = append(vouchers, *voucher)
	}

	token := CreateToken(sess, TokenParams{
		FromId: ident.UserId,
		FromAccount: ident.AccountId,
		FromName: ident.Username,
		To: "LastKeypair",
		Type: ident.Type,
		RemoteInstanceArn: instanceArn,
		Vouchers: vouchers,
		SshUsername: username,
	}, kmsKeyId)

	req := UserCertReqJson{
		EventType: "UserCertReq",
		Token: token,
		PublicKey: string(kp.PublicKey),
	}

	resp := UserCertRespJson{}
	err = RequestSignedPayload(sess, lambdaFunc, req, &resp)
	if err != nil {
		log.Panicf("err: %s", err.Error())
	}

	return req, resp
}

//func SshCommand(sess *session.Session, lambdaFunc, kmsKeyId, InstanceArn, username string, encodedVouchers, args []string) []string {
//	req, resp := sshReqResp(sess, lambdaFunc, kmsKeyId, InstanceArn, username, encodedVouchers)
//	return append(sshCommandFromResponse(req, resp), args...)
//}

type ReifiedLogin struct {
	sess            *session.Session
	lambdaFunc      string
	kmsKeyId        string
	InstanceArn     string
	username        string
	encodedVouchers []string
	args            []string

	Request  *UserCertReqJson
	Response *UserCertRespJson
}

func NewReifiedLoginWithCmd(cmd *cobra.Command, args []string) *ReifiedLogin {
	profile := viper.GetString("profile")

	lambdaFunc := viper.GetString("lambda-func")
	kmsKeyId := viper.GetString("kms-key")
	instanceArn, _ := cmd.PersistentFlags().GetString("instance-arn")
	username, _ := cmd.PersistentFlags().GetString("ssh-username")
	region, _ := cmd.PersistentFlags().GetString("region")
	vouchers, _ := cmd.PersistentFlags().GetStringSlice("voucher")
	
	instanceArnParts := strings.Split(instanceArn, ":")
	if len(instanceArnParts) > 3 {
		region = instanceArnParts[3]
	}
	sess := ClientAwsSession(profile, region)

	return &ReifiedLogin{
		sess:            sess,
		lambdaFunc:      lambdaFunc,
		kmsKeyId:        kmsKeyId,
		InstanceArn:     instanceArn,
		username:        username,
		encodedVouchers: vouchers,
		args:            args,
	}
}

func (r *ReifiedLogin) PopulateByInvoke() {
	req, resp := sshReqResp(r.sess, r.lambdaFunc, r.kmsKeyId, r.InstanceArn, r.username, r.encodedVouchers)

	r.Request = &req
	r.Response = &resp

	certPath := r.CertificatePath()
	ioutil.WriteFile(certPath, []byte(resp.SignedPublicKey), 0644)
	for _, j := range r.Response.Jumpboxes {
		ioutil.WriteFile(j.JumpCertificatePath(), []byte(j.SignedPublicKey), 0644)
	}

	serialized, _ := json.MarshalIndent(r, "", "  ")
	ioutil.WriteFile(r.Filepath("conn.json"), serialized, 0644)
}

func (r* ReifiedLogin) Filepath(name string) string {
	arn := r.InstanceArn
	arn = strings.Replace(arn, ":", "-", -1)
	arn = strings.Replace(arn, "/", "-", -1)
	arnDir := filepath.Join(TmpDir(), arn)
	os.MkdirAll(arnDir, 0755)
	return filepath.Join(arnDir, name)
}

func (j* Jumpbox) JumpboxFilepath() string {
	arn := j.HostKeyAlias
	arn = strings.Replace(arn, ":", "-", -1)
	arn = strings.Replace(arn, "/", "-", -1)
	arnDir := filepath.Join(TmpDir(), arn)
	os.MkdirAll(arnDir, 0755)
	return filepath.Join(arnDir)
}

func (r *ReifiedLogin) PopulateByRestoreCache() {
	serialized, _ := ioutil.ReadFile(r.Filepath("conn.json"))
	json.Unmarshal(serialized, r)
}

func (r *ReifiedLogin) WriteSshConfig() string {
	jump := r.Response.Jumpboxes

	filebuf := "IgnoreUnknown CertificateFile\n" // CertificateFile was introduced in 7.1

	for idx, j := range jump {
		filebuf = filebuf + fmt.Sprintf(`
Host jump%d
  HostName %s
  HostKeyAlias %s
  IdentityFile %s
  CertificateFile %s
  User %s
`, idx, j.Address, j.HostKeyAlias, r.PrivateKeyPath(), j.JumpCertificatePath(), j.User)
		if idx > 0 {
			filebuf = filebuf + fmt.Sprintf("  ProxyJump jump%d\n\n", idx-1)
		}
	}

	filebuf = filebuf + fmt.Sprintf(`
Host target
  HostKeyAlias %s
  IdentityFile %s
  CertificateFile %s
  User %s
`, r.Request.Token.Params.RemoteInstanceArn, r.PrivateKeyPath(), r.CertificatePath(), r.Request.Token.Params.SshUsername)

	if len(r.Response.TargetAddress) > 0 {
		filebuf = filebuf + fmt.Sprintf("  HostName %s\n", r.Response.TargetAddress)
	}

	if len(jump) > 0 {
		filebuf = filebuf + fmt.Sprintf("  ProxyJump jump%d\n\n", len(jump) - 1)
	}

	sshconfPath := r.Filepath("sshconf")
	ioutil.WriteFile(sshconfPath, []byte(filebuf), 0700)

	return sshconfPath
}

func (r *ReifiedLogin) PrivateKeyPath() string {
	return filepath.Join(AppDir(), "id_rsa")
}

func (r *ReifiedLogin) CertificatePath() string {
	return filepath.Join(AppDir(), "id_rsa-cert.pub")
}

func (j* Jumpbox) JumpCertificatePath() string {
	return filepath.Join(j.JumpboxFilepath(), "id_rsa-cert.pub")
}

func lambdaClientForKeyId(sess *session.Session, lambdaArn string) *lambda.Lambda {
	if strings.HasPrefix(lambdaArn, "arn:aws:lambda") {
		parts := strings.Split(lambdaArn, ":")
		region := parts[3]
		sess = sess.Copy(aws.NewConfig().WithRegion(region))
	}

	return lambda.New(sess)
}

func RequestSignedPayload(sess *session.Session, lambdaArn string, req interface{}, resp interface{}) error {
	ca := lambdaClientForKeyId(sess, lambdaArn)

	reqPayload, err := json.Marshal(&req)
	if err != nil {
		return errors.Wrap(err, "marshalling lambda req payload")
	}

	input := lambda.InvokeInput{
		FunctionName: aws.String(lambdaArn),
		Payload: reqPayload,
	}

	lambdaResp, err := ca.Invoke(&input)
	if err != nil {
		return errors.Wrap(err, "invoking CA lambda")
	}
	if lambdaResp.FunctionError != nil {
		return errors.New(fmt.Sprintf("%s: %s", *lambdaResp.FunctionError, string(lambdaResp.Payload)))
	}

	err = json.Unmarshal(lambdaResp.Payload, resp)
	if err != nil {
		return errors.Wrap(err, "unmarshalling lambda resp payload")
	}

	return nil
}
