#!/bin/bash
set -euxo pipefail

S3_BUCKET=lkp-lambda-test
S3_KEY=handler.zip

aws s3 cp handler.zip s3://$S3_BUCKET/$S3_KEY
S3_OBJVER=$(aws s3api head-object --bucket $S3_BUCKET --key $S3_KEY --query VersionId --output text)

./stackit \
  --stack-name lkp-lambda-test \
  --template cfn.yml \
  --previous-param-value PstoreCAKeyBytesName
  --previous-param-value S3Bucket
  --previous-param-value S3Key
  --param-value S3ObjectVersion=$S3_OBJVER

go run main.go ssh exec | tee out.log
diff out.log ci/expected-output.txt
