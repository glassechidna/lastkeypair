package common

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/kms"
	"encoding/json"
	"time"
	"github.com/pkg/errors"
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"os"
	"strconv"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/aws"
	"encoding/base64"
	"fmt"
	"log"
	"golang.org/x/crypto/ssh"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type UserCertReqJson struct {
	EventType string
	Token Token
	SshUsername string
	PublicKey string
}

type HostCertReqJson struct {
	EventType string
	Token Token
	PublicKey string
}

type UserCertRespJson struct {
	SignedPublicKey string
	Jumpbox *Jumpbox `json:",omitempty"`
	Expiry int64
}

type Jumpbox struct {
	IpAddress string
	InstanceId string
	User string
}

type HostCertRespJson struct {
	SignedHostPublicKey string
}

type AuthorizationLambdaIdentity struct {
	Name    *string `json:",omitempty"`
	Id      string
	Account string
	Type    string
}

type AuthorizationLambdaVoucher struct {
	Name    *string `json:",omitempty"`
	Id      string
	Account string
	Type    string
	Vouchee	string
	Context string
}

type AuthorizationLambdaRequest struct {
	From AuthorizationLambdaIdentity
	RemoteInstanceArn string
	Vouchers []AuthorizationLambdaVoucher `json:",omitempty"`
}

type AuthorizationLambdaResponse struct {
	Authorized bool
	Jumpbox *Jumpbox `json:",omitempty"`
	CertificateOptions struct {
		ForceCommand *string `json:",omitempty"`
		SourceAddress *string `json:",omitempty"`
	}
}

type LambdaConfig struct {
	KeyId string
	KmsTokenIdentity string
	CaKeyBytes []byte
	ValidityDuration int64
	AuthorizationLambda string
}

func getCaKeyBytes() ([]byte, error) {
	var caKeyBytes []byte

	if pstoreName, found := os.LookupEnv("PSTORE_CA_KEY_BYTES"); found {
		ssmClient := ssm.New(session.New())
		ssmInput := &ssm.GetParametersInput{
			Names: aws.StringSlice([]string{pstoreName}),
			WithDecryption: aws.Bool(true),
		}

		ssmResp, err := ssmClient.GetParameters(ssmInput)
		if err != nil {
			return nil, errors.Wrap(err, "decrypting key bytes from pstore")
		}

		valstr := ssmResp.Parameters[0].Value
		caKeyBytes = []byte(*valstr)
	} else if kmsEncrypted, found := os.LookupEnv("KMS_B64_CA_KEY_BYTES"); found {
		kmsClient := kms.New(session.New())

		b64dec, err := base64.StdEncoding.DecodeString(kmsEncrypted)
		if err != nil {
			return nil, errors.Wrap(err, "base64 decoding kms-encrypted ca key bytes")
		}

		kmsInput := &kms.DecryptInput{CiphertextBlob: b64dec}
		kmsResp, err := kmsClient.Decrypt(kmsInput)
		if err != nil {
			return nil, errors.Wrap(err, "decrypting kms-encrypted ca key bytes")
		}

		caKeyBytes = kmsResp.Plaintext
	} else if raw, found := os.LookupEnv("CA_KEY_BYTES"); found {
		caKeyBytes = []byte(raw)
	} else {
		return nil, errors.New("no ca key bytes provided")
	}

	return caKeyBytes, nil
}

func LambdaHandle(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	caKeyBytes, err := getCaKeyBytes()
	if err != nil {
		return nil, err
	}

	validity, err := strconv.ParseInt(os.Getenv("VALIDITY_DURATION"), 10, 64)

	config := LambdaConfig{
		KeyId: os.Getenv("KMS_KEY_ID"),
		KmsTokenIdentity: os.Getenv("KMS_TOKEN_IDENTITY"),
		CaKeyBytes: caKeyBytes,
		ValidityDuration: validity,
		AuthorizationLambda: os.Getenv("AUTHORIZATION_LAMBDA"),
	}

	raw := make(map[string]string)
	json.Unmarshal(evt, &raw)

	switch raw["EventType"] {
	case "UserCertReq":
		req := UserCertReqJson{}
		err := json.Unmarshal(evt, &req)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalling input")
		}
		return DoUserCertReq(req, config)
	case "HostCertReq":
		req := HostCertReqJson{}
		err := json.Unmarshal(evt, &req)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalling input")
		}
		return DoHostCertReq(req, config)
	default:
		return nil, errors.New("unexpected event type")
	}
}

func LambdaAwsSession() *session.Session {
	sessOpts := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		log.Panicf("couldn't create aws session")
	}

	return sess
}

func DoHostCertReq(req HostCertReqJson, config LambdaConfig) (*HostCertRespJson, error) {
	sess := LambdaAwsSession()

	if !ValidateToken(sess, req.Token, config.KeyId) {
		return nil, errors.New("invalid token")
	}

	permissions := ssh.Permissions{
		CriticalOptions: map[string]string{},
		Extensions: map[string]string{},
	}

	principal := req.Token.Params.HostInstanceArn
	signed, err := SignSsh(
		config.CaKeyBytes,
		[]byte(req.PublicKey),
		ssh.HostCert,
		ssh.CertTimeInfinity,
		permissions,
		principal,
		[]string{principal},
	)

	if err != nil {
		return nil, errors.Wrap(err, "signing ssh key")
	}

	resp := HostCertRespJson{
		SignedHostPublicKey: *signed,
	}

	return &resp, nil
}

func DoUserCertReq(req UserCertReqJson, config LambdaConfig) (*UserCertRespJson, error) {
	sess := LambdaAwsSession()

	if !ValidateToken(sess, req.Token, config.KeyId) {
		return nil, errors.New("invalid token")
	}

	identity := req.Token.Params.FromId
	if name := req.Token.Params.FromName; len(name) > 0 {
		identity = fmt.Sprintf("%s-%s", name, identity)
	}

	instanceArn := req.Token.Params.RemoteInstanceArn
	if len(instanceArn) == 0 {
		return nil, errors.New("target instance arn must be specified")
	}

	auth, err := DoAuthorizationLambda(req, config)
	if err != nil {
		return nil, errors.Wrap(err, "authorising user cert")
	}

	if !auth.Authorized {
		return nil, errors.New("authorisation denied by auth lambda")
	}

	principals := []string{instanceArn}
	if auth.Jumpbox != nil {
		principals = append(principals, auth.Jumpbox.InstanceId)
	}

	permissions := DefaultSshPermissions
	if auth.CertificateOptions.ForceCommand != nil {
		permissions.Extensions["force-command"] = *auth.CertificateOptions.ForceCommand
	}
	if auth.CertificateOptions.SourceAddress != nil {
		permissions.Extensions["source-address"] = *auth.CertificateOptions.SourceAddress
	}

	signed, err := SignSsh(
		config.CaKeyBytes,
		[]byte(req.PublicKey),
		ssh.UserCert,
		uint64(time.Now().Unix() + config.ValidityDuration),
		DefaultSshPermissions,
		identity,
		principals,
	)

	if err != nil {
		return nil, errors.Wrap(err, "signing ssh key")
	}

	expiry := time.Now().Add(time.Duration(config.ValidityDuration) * time.Second)

	resp := UserCertRespJson{
		SignedPublicKey: *signed,
		Jumpbox: auth.Jumpbox,
		Expiry: expiry.Unix(),
	}

	return &resp, nil
}

func DoAuthorizationLambda(userReq UserCertReqJson, config LambdaConfig) (*AuthorizationLambdaResponse, error) {
	if len(config.AuthorizationLambda) == 0 {
		return &AuthorizationLambdaResponse{Authorized: true}, nil
	}

	client := lambda.New(LambdaAwsSession())

	p := userReq.Token.Params
	req := AuthorizationLambdaRequest{
		From: AuthorizationLambdaIdentity{
			Name: &p.FromName,
			Id: p.FromId,
			Account: p.FromAccount,
			Type: p.Type,
		},
		RemoteInstanceArn: p.RemoteInstanceArn,
	}

	for _, v := range p.Vouchers {
		vp := v.Params
		voucher := AuthorizationLambdaVoucher{
			Name: &vp.FromName,
			Id: vp.FromId,
			Account: vp.FromAccount,
			Type: vp.Type,
			Vouchee: vp.Vouchee,
			Context: vp.Context,
		}
		req.Vouchers = append(req.Vouchers, voucher)
	}

	encoded, err := json.Marshal(&req)
	if err != nil {
		return nil, errors.Wrap(err, "encoding authorisation lambda request")
	}

	input := &lambda.InvokeInput{
		FunctionName: &config.AuthorizationLambda,
		Payload: encoded,
	}

	resp, err := client.Invoke(input)
	if err != nil {
		return nil, errors.Wrap(err, "executing authorisation lambda")
	}

	authResp := AuthorizationLambdaResponse{}
	err = json.Unmarshal(resp.Payload, &authResp)
	if err != nil {
		return nil, errors.Wrap(err, "decoding auth lambda response")
	}

	return &authResp, nil
}
