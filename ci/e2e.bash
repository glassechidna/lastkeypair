#!/bin/bash
set -euxo pipefail

export PATH=$PATH:$HOME/.local/bin # for awscli

S3_BUCKET=lkp-lambda-test
S3_KEY=handler.zip

aws s3 cp handler.zip s3://$S3_BUCKET/$S3_KEY
S3_OBJVER=$(aws s3api head-object --bucket $S3_BUCKET --key $S3_KEY --query VersionId --output text)

./stackit up \
  --stack-name lkp-lambda-test \
  --template ci/cfn.yml \
  --previous-param-value PstoreCAKeyBytesName \
  --previous-param-value S3Bucket \
  --previous-param-value S3Key \
  --param-value S3ObjectVersion=$S3_OBJVER

CONTAINER_ID=$(docker run -d glassechidna/sshello)
CONTAINER_IP=$(docker inspect $CONTAINER_ID --format '{{ .NetworkSettings.IPAddress }}')

docker run -it -v $(pwd):$(pwd) -w $(pwd) -e AWS_ACCESS_KEY_ID -e AWS_REGION -e AWS_SECRET_ACCESS_KEY rastasheep/ubuntu-sshd:16.04 ./lastkeypair_linux_amd64 ssh exec -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 travis@$CONTAINER_IP | tee out.log
#go run main.go ssh exec -- -o StrictHostKeyChecking=no -p 2222 travis@$CONTAINER_IP | tee out.log
diff out.log ci/expected-output.txt
