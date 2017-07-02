#!/bin/bash
set -euxo pipefail

go get github.com/mitchellh/gox

curl -z upx.txz -o upx.txz -L https://github.com/upx/upx/releases/download/v3.93/upx-3.93-amd64_linux.tar.xz
tar -xvf upx.txz

curl -z glide.tgz -o glide.tgz -L https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz
tar -xvf glide.tgz

curl -z stackit -o stackit -L https://github.com/glassechidna/stackit/releases/download/0.0.9/stackit_linux_amd64
chmod +x stackit
curl -z dl/sshello -o dl/sshello -L https://github.com/glassechidna/sshello/releases/download/0.0.1/sshello_linux_amd64
chmod +x dl/sshello

curl -z ci/shim/runtime.so -o ci/shim/runtime.so -L https://github.com/glassechidna/lastkeypair/releases/download/0.0.1/runtime.so

pip install --user awscli
