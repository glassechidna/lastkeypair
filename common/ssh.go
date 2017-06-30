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
	"github.com/aws/aws-sdk-go/service/sts"
	"os/exec"
	"syscall"
	"os"
)

func SshExec(sess *session.Session, lambdaFunc, funcIdentity, kmsKeyId string, args []string) {
	kp, _ := MyKeyPair()

	stsClient := sts.New(sess)
	stsAcct, stsFrom, err := CallerIdentityUser(stsClient)
	if err != nil {
		log.Panicf("error getting aws user identity: %+v\n", err)
	}

	token := CreateToken(sess, TokenParams{
		KeyId: kmsKeyId,
		From: *stsFrom,
		FromAccount: *stsAcct,
		To: funcIdentity,
		Type: "user",
	})

	req := UserCertReqJson{
		EventType: "UserCertReq",
		Token: token,
		InstanceId: "",
		PublicKey: string(kp.PublicKey),
	}

	signed, err := RequestSignedCert(sess, lambdaFunc, req)
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
		"CertificateFile=/root/.lkp/id_rsa-cert.pub",
	}

	args = append(lkpArgs, args...)
	sshPath, _ := exec.LookPath("ssh")
	syscall.Exec(sshPath, args, os.Environ())
}

func RequestSignedCert(sess *session.Session, lambdaArn string, req UserCertReqJson) (*UserCertRespJson, error) {
	ca := lambda.New(sess)

	reqPayload, err := json.Marshal(&req)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling lambda req payload")
	}

	input := lambda.InvokeInput{
		FunctionName: aws.String(lambdaArn),
		Payload: reqPayload,
	}

	resp, err := ca.Invoke(&input)
	if err != nil {
		return nil, errors.Wrap(err, "invoking CA lambda")
	}

	payload := UserCertRespJson{}
	err = json.Unmarshal(resp.Payload, &payload)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling lambda resp payload")
	}

	return &payload, nil
}

