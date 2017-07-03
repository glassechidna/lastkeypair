package common

import (
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws"
	"time"
	"encoding/json"
	"log"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/service/sts"
	"strings"
	"golang.org/x/crypto/ssh"
	"fmt"
	"github.com/pkg/errors"
	"crypto/rand"
	"github.com/aws/aws-sdk-go/aws/request"
)

var ApplicationVersion string
var ApplicationBuildDate string

func SignSsh(caKeyBytes, userPubkeyBytes []byte, durationSecs int64, keyId string, principals []string) (*string, error) {
	signer, err := ssh.ParsePrivateKey(caKeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "err parsing ca priv key")
	}

	userPubkey, _, _, _, err := ssh.ParseAuthorizedKey(userPubkeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "err parsing user pub key")
	}

	now := time.Now()
	after := now.Add(-300 * time.Second)
	before := now.Add(time.Duration(durationSecs) * time.Second)

	cert := &ssh.Certificate{
		//Nonce: is generated by cert.SignCert
		Key: userPubkey,
		Serial: 0,
		CertType: ssh.UserCert,
		KeyId: keyId,
		ValidPrincipals: principals,
		ValidAfter: uint64(after.Unix()),
		ValidBefore: uint64(before.Unix()),
		Permissions: ssh.Permissions{
			CriticalOptions: map[string]string{},
			Extensions: map[string]string{
				"permit-X11-forwarding":   "",
				"permit-agent-forwarding": "",
				"permit-port-forwarding":  "",
				"permit-pty":              "",
				"permit-user-rc":          "",
			},
		},
		Reserved: []byte{},
	}

	randSource := rand.Reader
	err = cert.SignCert(randSource, signer)
	if err != nil {
		return nil, errors.Wrap(err, "err signing cert")
	}

	signed := cert.Marshal()

	b64 := base64.StdEncoding.EncodeToString(signed)
	formatted := fmt.Sprintf("%s %s", cert.Type(), b64)
	return &formatted, nil
}

func AwsSession(profile, region string) *session.Session {
	sessOpts := session.Options{
		SharedConfigState: session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}

	if len(profile) > 0 {
		sessOpts.Profile = profile
	}

	sess, _ := session.NewSessionWithOptions(sessOpts)
	config := aws.NewConfig()

	userAgentHandler := request.NamedHandler{
		Name: "LastKeypair.UserAgentHandler",
		Fn:   request.MakeAddToUserAgentHandler("LastKeypair", ApplicationVersion),
	}
	sess.Handlers.Build.PushBackNamed(userAgentHandler)

	if len(region) > 0 {
		config.Region = aws.String(region)
		sess.Config = config
	}

	return sess
}

type PlaintextPayload struct {
	NotBefore float64 // this is what json.unmarshal wants
	NotAfter float64
}

type TokenParams struct {
	KeyId string
	FromId string
	FromAccount string
	FromName string
	To string
	Type string
}

func (params *TokenParams) ToKmsContext() map[string]*string {
	context := make(map[string]*string)
	context["fromId"] = &params.FromId
	context["fromAccount"] = &params.FromAccount
	context["to"] = &params.To
	context["type"] = &params.Type

	if len(params.FromName) > 0 {
		context["fromName"] = &params.FromName
	}

	return context
}

type Token struct {
	Params TokenParams
	Signature []byte
}

func CreateToken(sess *session.Session, params TokenParams) Token {
	context := params.ToKmsContext()

	now := float64(time.Now().Unix())
	end := now + 3600 // 1 hour

	payload := PlaintextPayload{
		NotBefore: now,
		NotAfter: end,
	}

	plaintext, err := json.Marshal(&payload)
	if err != nil {
		log.Panicf("Payload json encoding error: %s", err.Error())
	}

	input := &kms.EncryptInput{
		Plaintext: plaintext,
		KeyId: &params.KeyId,
		EncryptionContext: context,
	}

	client := kms.New(sess)
	response, err := client.Encrypt(input)
	if err != nil {
		log.Panicf("Encrytion error: %s", err.Error())
	}

	blob := response.CiphertextBlob
	params.KeyId = *response.KeyId
	return Token{Params: params, Signature: blob}
}

func ValidateToken(sess *session.Session, token Token, expectedKeyId string) bool {
	context := token.Params.ToKmsContext()

	input := &kms.DecryptInput{
		CiphertextBlob: token.Signature,
		EncryptionContext: context,
	}

	client := kms.New(sess)
	response, err := client.Decrypt(input)
	if err != nil {
		log.Panicf("Decryption error: %s", err.Error())
	}

	/* We verify that the encryption key used is the one that we expected it to be.
	   This is very important, as an attacker could submit ciphertext encrypted with
	   a key they control that grants our Lambda permission to decrypt. Perhaps it
	   would be worth implementing some kind of alert here?
	 */
	if expectedKeyId != *response.KeyId {
		log.Panicf("Mismatching KMS key ids: %s and %s", token.Params.KeyId, *response.KeyId)
	}

	payload := PlaintextPayload{}
	err = json.Unmarshal([]byte(response.Plaintext), &payload)
	if err != nil {
		return false
		//return nil, errors.Wrap(err, "decoding token json")
	}

	now := float64(time.Now().Unix())
	if now < payload.NotBefore || now > payload.NotAfter {
		return false
		//return nil, errors.New("expired token")
	}

	return true
}

type StsIdentity struct {
	AccountId string
	UserId string
	Username string
	Type string
}

func CallerIdentityUser(sess *session.Session) (*StsIdentity, error) {
	client := sts.New(sess)
	response, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err == nil {
		arn := *response.Arn
		parts := strings.SplitN(arn, ":", 6)

		if strings.HasPrefix(parts[5], "user/") {
			name := parts[5][5:]
			return &StsIdentity{
				AccountId: *response.Account,
				UserId: *response.UserId,
				Username: name,
				Type: "User",
			}, nil
		} else if strings.HasPrefix(parts[5], "assumed-role/") {
			return &StsIdentity{
				AccountId: *response.Account,
				UserId: *response.UserId,
				Username: "",
				Type: "AssumedRole",
			}, nil
		} else {
			return nil, errors.New("unsupported IAM identity type")
		}
	} else {
		return nil, err
	}
}