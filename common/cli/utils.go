package cli

import (
	"strings"
	"github.com/aws/aws-sdk-go/aws/session"
	"regexp"
	"fmt"
	"github.com/pkg/errors"
)

func FullKmsKey(sess *session.Session, input string) (string, error) {
	myRegion := *sess.Config.Region

	if strings.HasPrefix(input, "arn:aws:kms") { return input, nil }

	regex, _ := regexp.Compile("(\\d+):(.+)")
	matches := regex.FindStringSubmatch(input)
	if len(matches) == 3 {
		if len(myRegion) == 0 {
			return "", errors.New("can't deduce key arn without a valid region set")
		}
		accountId := matches[1]
		keyOrAlias := matches[2]
		return fmt.Sprintf("arn:aws:kms:%s:%s:%s", myRegion, accountId, keyOrAlias), nil
	} else {
		return input, nil
	}

	return "", nil
}

