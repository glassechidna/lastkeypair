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
  --previous-param-value JumpboxDns \
  --param-value S3ObjectVersion=$S3_OBJVER

./dl/sshello &
sleep 1
./lkp_linux_amd64 ssh exec --kms-key $AWS_ACCOUNT_ID:alias/LastKeypair --instance-arn abcdef -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 -o HostName=localhost travis@target | tee out.log
diff out.log ci/expected-output.txt

./lkp_linux_amd64 ssh exec --instance-arn defghi --dry-run -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 | tee out.log
diff out.log ci/expected-output-jumpbox.txt
diff ~/.lkp/tmp/defghi/sshconf ci/expected-output-jumpbox-sshconf.txt

VOUCHER=$(./lkp_linux_amd64 adv vouch --vouchee aidan --context moo)
./lkp_linux_amd64 ssh exec --instance-arn defghi --voucher $VOUCHER --dry-run -- -o StrictHostKeyChecking=no -o LogLevel=QUIET -p 2222 travis@localhost | tee out.log
diff out.log ci/expected-output-vouched.txt

# openssl aes-256-cbc -pass "pass:$CI_SSH_PASSPHRASE" -in ci/ec2-ssh-key.enc -out ci/ec2-ssh-key -d -a
# chmod 0600 ci/ec2-ssh-key
# cat ci/ec2-ssh-host-key >> ~/.ssh/known_hosts

# scp -i ci/ec2-ssh-key lkp_linux_amd64 ec2-user@$GE_CI_HOST:~/

# ssh -T -i ci/ec2-ssh-key ec2-user@$GE_CI_HOST << 'ENDSSH' | tee out.log
#     rm -rf lkp-ci
#     mkdir lkp-ci

#     cp /etc/ssh/ssh_host_rsa_key.pub lkp-ci/ssh_host_rsa_key.pub

#     touch \
#       lkp-ci/authorized_principals \
#       lkp-ci/cert_authority.pub \
#       lkp-ci/ssh_host_rsa_key-cert.pub \
#       lkp-ci/sshd_config

#     ./lkp_linux_amd64 \
#       host \
#       --authorized-principals-path lkp-ci/authorized_principals \
#       --cert-authority-path        lkp-ci/cert_authority.pub \
#       --host-key-path              lkp-ci/ssh_host_rsa_key.pub \
#       --signed-host-key-path       lkp-ci/ssh_host_rsa_key-cert.pub \
#       --sshd-config-path           lkp-ci/sshd_config

#     cat lkp-ci/authorized_principals
#     cat lkp-ci/sshd_config
#     ssh-keygen -Lf lkp-ci/ssh_host_rsa_key-cert.pub
# ENDSSH
# diff -I 'Valid: after' out.log ci/expected-output-host.txt
