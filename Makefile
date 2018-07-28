HANDLER ?= handler
PACKAGE ?= $(HANDLER)
GOPATH  ?= $(HOME)/go

VERSION = $(shell git describe --tags)
DATE = $(shell date +%FT%T%z)
GO_LDFLAGS := "-X github.com/glassechidna/lastkeypair/pkg/lastkeypair.ApplicationVersion=$(VERSION) -X github.com/glassechidna/lastkeypair/pkg/lastkeypair.ApplicationBuildDate=$(DATE)"

all: linux otherplats zip

.PHONY: all

linux:
	@gox -arch="amd64" -os="linux" -ldflags=$(GO_LDFLAGS) ./cmd/...
	@upx lkp_linux_amd64

otherplats:
	@gox -arch="amd64" -os="windows darwin" -ldflags=$(GO_LDFLAGS) ./cmd/lkp
	@upx lkp_darwin_amd64 lkp_windows_amd64.exe

zip:
	@zip handler.zip lambda_linux_amd64
