#!/bin/bash

export PATH=$PATH:$HOME/.local/bin # for awscli

set -euxo pipefail

S3_BUCKET=lkp-lambda-test
S3_KEY=handler.zip

aws s3 cp handler.zip s3://$S3_BUCKET/$S3_KEY
S3_OBJVER=$(aws s3api head-object --bucket $S3_BUCKET --key $S3_KEY --query VersionId --output text)

./dl/stackit up \
  --stack-name lkp-lambda-test \
  --template ci/cfn.yml \
  --previous-param-value PstoreCAKeyBytesName \
  --previous-param-value S3Bucket \
  --previous-param-value S3Key \
  --previous-param-value AllowedAwsAccounts \
  --param-value S3ObjectVersion=$S3_OBJVER

./dl/sshello &
./lastkeypair_linux_amd64 ssh exec --instance-arn abcdef -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 travis@localhost | tee out.log
diff out.log ci/expected-output.txt

./lastkeypair_linux_amd64 ssh exec --instance-arn defghi --dry-run -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 travis@localhost | tee out.log
diff out.log ci/expected-output-jumpbox.txt

VOUCHER=$(./lastkeypair_linux_amd64 vouch --vouchee aidan --context moo)
./lastkeypair_linux_amd64 ssh exec --instance-arn defghi --voucher $VOUCHER --dry-run -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 travis@localhost | tee out.log
diff out.log ci/expected-output-vouched.txt
