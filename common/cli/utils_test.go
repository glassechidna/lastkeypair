package cli

import (
	"testing"
	"github.com/aws/aws-sdk-go/aws/session"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/aws"
)

func TestFullKmsKey(t *testing.T) {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))
	full := "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"

	a := ass{t: t}
	str := a.assertNoErr(FullKmsKey(sess, full))
	assertEqual(t, str, full, "")

	str = a.assertNoErr(FullKmsKey(sess, "123456789012:key/12345678-1234-1234-1234-123456789012"))
	assertEqual(t, str, full, "")

	str = a.assertNoErr(FullKmsKey(sess, "123456789012:alias/myalias"))
	assertEqual(t, str, full, "")
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	if len(message) == 0 {
		message = fmt.Sprintf("%v != %v", a, b)
	}
	t.Fatal(message)
}

type ass struct{
	t *testing.T
}

func (a *ass) assertNoErr(str string, err error) string {
	assert.Nil(a.t, err)
	return str
}
