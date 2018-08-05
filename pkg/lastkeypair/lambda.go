package lastkeypair

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/kms"
	"encoding/json"
	"time"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/aws"
	"encoding/base64"
	"fmt"
	"log"
	"golang.org/x/crypto/ssh"
)

type LambdaConfig struct {
	KeyId string
	KmsTokenIdentity string
	CaKeyBytes []byte
	CaKeyPassphraseBytes []byte
	ValidityDuration int64
	AuthorizationLambda string
}

func getPstoreOrKmsOrRawBytes(name string) ([]byte, error) {
	var bytes []byte

	if pstoreName, found := os.LookupEnv(fmt.Sprintf("PSTORE_%s", name)); found {
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
		bytes = []byte(*valstr)
	} else if kmsEncrypted, found := os.LookupEnv(fmt.Sprintf("KMS_B64_%s", name)); found {
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

		bytes = kmsResp.Plaintext
	} else if raw, found := os.LookupEnv(name); found {
		bytes = []byte(raw)
	} else {
		return nil, nil
	}

	return bytes, nil
}

func LambdaHandle(evt json.RawMessage) (interface{}, error) {
	caKeyBytes, err := getPstoreOrKmsOrRawBytes("CA_KEY_BYTES")
	if err != nil {
		return nil, err
	} else if caKeyBytes == nil {
		return nil, errors.New("no ca key bytes provided")
	}

	caKeyPassphraseBytes, err := getPstoreOrKmsOrRawBytes("CA_KEY_PASSPHRASE_BYTES")
	if err != nil {
		return nil, err
	}

	validity, err := strconv.ParseInt(os.Getenv("VALIDITY_DURATION"), 10, 64)

	kmsTokenIdentity := os.Getenv("KMS_TOKEN_IDENTITY")
	if len(kmsTokenIdentity) == 0 {
		kmsTokenIdentity = "LastKeypair"
	}

	config := LambdaConfig{
		KeyId: os.Getenv("KMS_KEY_ID"),
		KmsTokenIdentity: kmsTokenIdentity,
		CaKeyBytes: caKeyBytes,
		CaKeyPassphraseBytes: caKeyPassphraseBytes,
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

	authLambda := NewAuthorizationLambda(config)
	auth, err := authLambda.DoHostReq(req)

	signed, err := SignSsh(
		config.CaKeyBytes,
		config.CaKeyPassphraseBytes,
		[]byte(req.PublicKey),
		ssh.HostCert,
		ssh.CertTimeInfinity,
		permissions,
		auth.KeyId,
		auth.Principals,
	)

	if err != nil {
		return nil, errors.Wrap(err, "signing ssh key")
	}

	resp := HostCertRespJson{
		SignedHostPublicKey: *signed,
	}

	return &resp, nil
}

func GenerateSshPermissions(options *CertificateOptions) ssh.Permissions {
	var SshPermissions = ssh.Permissions{
		CriticalOptions: map[string]string{},
		Extensions: map[string]string{
			"permit-X11-forwarding":   "",
			"permit-agent-forwarding": "",
			"permit-port-forwarding":  "",
			"permit-pty":              "",
			"permit-user-rc":          "",
		},
	}

	if options.ForceCommand != nil {
		SshPermissions.Extensions["force-command"] = *options.ForceCommand
	}
	if options.SourceAddress != nil {
		SshPermissions.Extensions["source-address"] = *options.SourceAddress
	}
	if options.PermitX11Forwarding == false {
		delete(SshPermissions.Extensions, "permit-X11-forwarding")
	}
	if options.PermitAgentForwarding == false {
		delete(SshPermissions.Extensions, "permit-agent-forwarding")
	}
	if options.PermitPortForwarding == false {
		delete(SshPermissions.Extensions, "permit-port-forwarding")
	}

	return SshPermissions
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

	authLambda := NewAuthorizationLambda(config)
	auth, err := authLambda.DoUserReq(req)
	if err != nil {
		return nil, errors.Wrap(err, "authorising user cert")
	}

	if !auth.Authorized {
		errorMessage := "authorisation denied by auth lambda"
		if len(auth.Message) > 0 {
			errorMessage = auth.Message
		}
		return nil, errors.New(errorMessage)
	}

	SshPermissions := GenerateSshPermissions(auth.CertificateOptions)

	signed, err := SignSsh(
		config.CaKeyBytes,
		config.CaKeyPassphraseBytes,
		[]byte(req.PublicKey),
		ssh.UserCert,
		uint64(time.Now().Unix() + config.ValidityDuration),
		SshPermissions,
		identity,
		auth.Principals,
	)

	for idx := range auth.Jumpboxes {
		j := &auth.Jumpboxes[idx]
		if len(j.HostKeyAlias) == 0 {
			j.HostKeyAlias = j.Address
		}
		if len(j.Principals) == 0 {
			j.Principals = append(j.Principals, j.Address)
		}
		jSshPermissions := GenerateSshPermissions(j.CertificateOptions)
		jSigned, jErr := SignSsh(
			config.CaKeyBytes,
			config.CaKeyPassphraseBytes,
			[]byte(req.PublicKey),
			ssh.UserCert,
			uint64(time.Now().Unix() + config.ValidityDuration),
			jSshPermissions,
			identity,
			j.Principals,
		)
		j.SignedPublicKey = *jSigned
		if jErr != nil {
			return nil, errors.Wrap(err, "error signing ssh key for jumphost")
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "error signing ssh key")
	}

	expiry := time.Now().Add(time.Duration(config.ValidityDuration) * time.Second)

	resp := UserCertRespJson{
		SignedPublicKey: *signed,
		Jumpboxes: auth.Jumpboxes,
		TargetAddress: auth.TargetAddress,
		Expiry: expiry.Unix(),
	}

	return &resp, nil
}
