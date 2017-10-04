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
)

func SshCommand(sess *session.Session, lambdaFunc, funcIdentity, kmsKeyId, instanceArn, username string, encodedVouchers, args []string) []string {
	kp, _ := MyKeyPair()

	ident, err := CallerIdentityUser(sess)
	if err != nil {
		log.Panicf("error getting aws user identity: %+v\n", err)
	}

	vouchers := []VoucherToken{}
	for _, encVoucher := range(encodedVouchers) {
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
		To: funcIdentity,
		Type: ident.Type,
		RemoteInstanceArn: instanceArn,
		Vouchers: vouchers,
	}, kmsKeyId)

	req := UserCertReqJson{
		EventType: "UserCertReq",
		Token: token,
		SshUsername: username,
		PublicKey: string(kp.PublicKey),
	}

	signed := UserCertRespJson{}
	err = RequestSignedPayload(sess, lambdaFunc, req, &signed)
	if err != nil {
		log.Panicf("err: %s", err.Error())
	}

	certPath := path.Join(AppDir(), "id_rsa-cert.pub")
	ioutil.WriteFile(certPath, []byte(signed.SignedPublicKey), 0644)

	lkpArgs := []string{
		"ssh",
		"-o",
		"IdentityFile=~/.lkp/id_rsa",
	}

	if signed.Jumpbox != nil {
		proxyCommand := fmt.Sprintf("ProxyCommand='ssh -W %%h:%%p %s@%s'", signed.Jumpbox.User, signed.Jumpbox.IpAddress)
		lkpArgs = append(lkpArgs, "-o", proxyCommand)
	}

	args = append(lkpArgs, args...)
	return args
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

	err = json.Unmarshal(lambdaResp.Payload, resp)
	if err != nil {
		return errors.Wrap(err, "unmarshalling lambda resp payload")
	}

	return nil
}
