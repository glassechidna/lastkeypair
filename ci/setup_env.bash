#!/bin/bash
set -euxo pipefail

go get github.com/mitchellh/gox

curl -z dl/upx.txz -o dl/upx.txz -L https://github.com/upx/upx/releases/download/v3.94/upx-3.94-amd64_linux.tar.xz
tar -xvf dl/upx.txz

curl -z dl/stackit -o dl/stackit -L https://github.com/glassechidna/stackit/releases/download/0.0.9/stackit_linux_amd64
chmod +x dl/stackit

curl -z dl/sshello -o dl/sshello -L https://github.com/glassechidna/sshello/releases/download/0.0.1/sshello_linux_amd64
chmod +x dl/sshello

pip install --user awscli
