package common

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/kms"
	"encoding/json"
	"time"
	"io"
	"github.com/pkg/errors"
	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"os"
	"strconv"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/aws"
	"encoding/base64"
)

type UserCertReqJson struct {
	EventType string
	Token string
	From string
	To string
	Type string
	InstanceId string
	PublicKey string
}

type CaKeyBytesProvider interface {
	CaKeyBytes() []byte
}

type PstoreKeyBytesProvider struct {

}

type UserCertRespJson struct {
	SignedPublicKey string
	Expiry int64
}

type LambdaConfig struct {
	KeyId string
	KmsTokenIdentity string
	CaKeyBytes []byte
	ValidityDuration int64
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
	req := UserCertReqJson{}
	err := json.Unmarshal(evt, &req)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling input")
	}

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
	}

	resp, err := DoUserCertReq(req, config)
	return resp, err
}

func ParseInput(input io.Reader) (*UserCertReqJson, error) {
	req := UserCertReqJson{}
	err := json.NewDecoder(input).Decode(&req)
	if err != nil {
		return nil, errors.Wrap(err, "decoding stdin json")
	}

	return &req, nil
}

func DoUserCertReq(req UserCertReqJson, config LambdaConfig) (*UserCertRespJson, error) {
	sessOpts := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating aws session")
	}
	client := kms.New(sess)

	payload := PlaintextPayload{}
	payloadStr := ValidateToken(client, config.KeyId, req.From, config.KmsTokenIdentity, req.Type, req.Token)
	err = json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		return nil, errors.Wrap(err, "decoding token json")
	}

	now := float64(time.Now().Unix())
	if now < payload.NotBefore || now > payload.NotAfter {
		return nil, errors.New("expired token")
	}

	signed, err := SignSsh(config.CaKeyBytes, []byte(req.PublicKey), config.ValidityDuration, req.From, []string{})
	if err != nil {
		return nil, errors.Wrap(err, "signing ssh key")
	}

	expiry := time.Now().Add(time.Duration(config.ValidityDuration) * time.Second)

	resp := UserCertRespJson{
		SignedPublicKey: *signed,
		Expiry: expiry.Unix(),
	}

	return &resp, nil
}
