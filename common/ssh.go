package common

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"path"
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
	region, _ := cmd.PersistentFlags().GetString("region")
	sess := ClientAwsSession(profile, region)

	lambdaFunc := viper.GetString("lambda-func")
	kmsKeyId := viper.GetString("kms-key")
	instanceArn, _ := cmd.PersistentFlags().GetString("instance-arn")
	username, _ := cmd.PersistentFlags().GetString("ssh-username")
	vouchers, _ := cmd.PersistentFlags().GetStringSlice("voucher")

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

	certPath := r.CertificatePath()
	ioutil.WriteFile(certPath, []byte(resp.SignedPublicKey), 0644)

	r.Request = &req
	r.Response = &resp

	serialized, _ := json.MarshalIndent(r, "", "  ")
	ioutil.WriteFile(r.SerializedPath(), serialized, 0644)
}

func (r *ReifiedLogin) SerializedPath() string {
	// make name filesystem-friendly
	arn := strings.Replace(r.InstanceArn, ":", "-", -1)
	arn = strings.Replace(arn, "/", "-", -1)
	return path.Join(AppDir(), fmt.Sprintf("conn-%s.json", arn))
}

func (r *ReifiedLogin) PopulateByRestoreCache() {
	serialized, _ := ioutil.ReadFile(r.SerializedPath())
	json.Unmarshal(serialized, r)
}

func (r *ReifiedLogin) WriteSshConfig() string {
	jump := r.Response.Jumpboxes

	sshconfPath := filepath.Join(AppDir(), "sshconf")
	f, err := os.OpenFile(sshconfPath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil { panic(err) }

	for idx, j := range jump {
		f.WriteString(fmt.Sprintf(`
Host jump%d
  HostName %s
  HostKeyAlias %s
  IdentityFile %s
  CertificateFile %s
  User %s
`, idx, j.Address, j.HostKeyAlias, r.PrivateKeyPath(), r.CertificatePath(), j.User))
		if idx > 0 {
			f.WriteString(fmt.Sprintf("  ProxyJump jump%d\n\n", idx-1))
		}
	}

	f.WriteString(fmt.Sprintf(`
Host target
  HostName %s
  HostKeyAlias %s
  IdentityFile %s
  CertificateFile %s
  User %s
`, r.Response.TargetAddress, r.Request.Token.Params.RemoteInstanceArn, r.PrivateKeyPath(), r.CertificatePath(), r.Request.Token.Params.SshUsername))

	if len(jump) > 0 {
		f.WriteString(fmt.Sprintf("  ProxyJump jump%d\n\n", len(jump) - 1))
	}

	f.Close()
	return sshconfPath
}

func (r *ReifiedLogin) PrivateKeyPath() string {
	return filepath.Join(AppDir(), "id_rsa")
}

func (r *ReifiedLogin) CertificatePath() string {
	return filepath.Join(AppDir(), "id_rsa-cert.pub")
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
