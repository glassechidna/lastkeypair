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

func SshCommand(sess *session.Session, lambdaFunc, kmsKeyId, instanceArn, username string, encodedVouchers, args []string) []string {
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
		"-o",
		fmt.Sprintf("HostKeyAlias=%s", instanceArn),
	}

	if len(signed.Jumpboxes) > 0 {
		jumps := []string{}
		for _, jbox := range signed.Jumpboxes {
			jumps = append(jumps, fmt.Sprintf("%s@%s", jbox.User, jbox.Address))
		}
		joinedJumps := strings.Join(jumps, ",")
		lkpArgs = append(lkpArgs, "-J", joinedJumps)
	}

	args = append(lkpArgs, args...)

	if len(signed.TargetAddress) > 0 {
		args = append(args, fmt.Sprintf("%s@%s", username, signed.TargetAddress))
	}

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
	if lambdaResp.FunctionError != nil {
		return errors.New(fmt.Sprintf("%s: %s", *lambdaResp.FunctionError, string(lambdaResp.Payload)))
	}

	err = json.Unmarshal(lambdaResp.Payload, resp)
	if err != nil {
		return errors.Wrap(err, "unmarshalling lambda resp payload")
	}

	return nil
}
