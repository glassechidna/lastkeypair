#!/bin/bash
set -euxo pipefail

export PATH=$PATH:$HOME/.local/bin # for awscli

S3_BUCKET=lkp-lambda-test
S3_KEY=handler.zip

make # build lambda package
aws s3 cp handler.zip s3://$S3_BUCKET/$S3_KEY
S3_OBJVER=$(aws s3api head-object --bucket $S3_BUCKET --key $S3_KEY --query VersionId --output text)

./stackit up \
  --stack-name lkp-lambda-test \
  --template ci/cfn.yml \
  --previous-param-value PstoreCAKeyBytesName
  --previous-param-value S3Bucket
  --previous-param-value S3Key
  --param-value S3ObjectVersion=$S3_OBJVER

CONTAINER_ID=$(docker run -d glassechidna/sshello)
CONTAINER_IP=$(docker inspect $CONTAINER_ID --format '{{ .NetworkSettings.IPAddress }}')

go run main.go ssh exec -- travis@$CONTAINER_IP | tee out.log
diff out.log ci/expected-output.txt
