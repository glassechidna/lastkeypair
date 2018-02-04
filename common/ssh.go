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
)

func sshCommandFromResponse(req UserCertReqJson, resp UserCertRespJson) []string {
	lkpArgs := []string{
		"ssh",
		"-o",
		"IdentityFile=~/.lkp/id_rsa",
		"-o",
		fmt.Sprintf("HostKeyAlias=%s", req.Token.Params.RemoteInstanceArn),
	}

	if len(resp.Jumpboxes) > 0 {
		jumps := []string{}
		for _, jbox := range resp.Jumpboxes {
			jumps = append(jumps, fmt.Sprintf("%s@%s", jbox.User, jbox.Address))
		}
		joinedJumps := strings.Join(jumps, ",")
		lkpArgs = append(lkpArgs, "-J", joinedJumps)
	}

	if len(resp.TargetAddress) > 0 {
		lkpArgs = append(lkpArgs, "-W", resp.TargetAddress + ":22")
	}

	return lkpArgs
}

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

	certPath := path.Join(AppDir(), "id_rsa-cert.pub")
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

func (r *ReifiedLogin) SshCommand() []string {
	return append(sshCommandFromResponse(*r.Request, *r.Response), r.args...)
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
