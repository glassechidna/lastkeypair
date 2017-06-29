#!/bin/bash
set -euxo pipefail

go get github.com/mitchellh/gox

curl -z upx.txz -o upx.txz -L https://github.com/upx/upx/releases/download/v3.93/upx-3.93-amd64_linux.tar.xz
tar -xvf upx.txz

curl -z glide.tgz -o glide.tgz -L https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz
tar -xvf glide.tgz

curl -z stackit -o stackit -L https://github.com/glassechidna/stackit/releases/download/0.0.9/stackit_linux_amd64
chmod +x stackit

pip install --user awscli
