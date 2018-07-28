package lastkeypair

import (
	"encoding/json"
	"bytes"
	"compress/gzip"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
	"encoding/base32"
	"github.com/pkg/errors"
	"io/ioutil"
)

type VoucherToken Token

func (vt *VoucherToken) Encode() string {
	jsonToken, _ := json.Marshal(vt)
	buf := bytes.Buffer{}
	gz := gzip.NewWriter(&buf)
	gz.Write(jsonToken)
	gz.Flush()
	gz.Close()
	encoded := base32.StdEncoding.EncodeToString(buf.Bytes())
	return encoded
}

func DecodeVoucherToken(encoded string) (*VoucherToken, error) {
	compressed, err := base32.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.Wrap(err, "decoding base32 voucher")
	}

	bytesReader := bytes.NewReader(compressed)
	gz, err := gzip.NewReader(bytesReader)
	if err != nil {
		return nil, errors.Wrap(err, "creating gzip reader for voucher")
	}

	jsonToken, err := ioutil.ReadAll(gz)
	if err != nil {
		return nil, errors.Wrap(err, "reading gzipped voucher")
	}

	token := VoucherToken{}
	err = json.Unmarshal(jsonToken, &token)
	if err != nil {
		return nil, errors.Wrap(err, "reading json voucher")
	}

	return &token, nil
}

func Vouch(sess *session.Session, kmsKeyId, to, vouchee, context string) VoucherToken {
	ident, err := CallerIdentityUser(sess)
	if err != nil {
		log.Panicf("error getting aws user identity: %+v\n", err)
	}

	token := CreateToken(sess, TokenParams{
		FromId: ident.UserId,
		FromAccount: ident.AccountId,
		FromName: ident.Username,
		To: to,
		Type: ident.Type,
		Vouchee: vouchee,
		Context: context,
	}, kmsKeyId)

	return VoucherToken(token)
}

