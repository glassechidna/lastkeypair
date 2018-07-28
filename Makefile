HANDLER ?= handler
PACKAGE ?= $(HANDLER)
GOPATH  ?= $(HOME)/go

VERSION = $(shell git describe --tags)
DATE = $(shell date +%FT%T%z)
GO_LDFLAGS := "-X github.com/glassechidna/lastkeypair/pkg/lastkeypair.ApplicationVersion=$(VERSION) -X github.com/glassechidna/lastkeypair/pkg/lastkeypair.ApplicationBuildDate=$(DATE)"

all: linux build pack perm

.PHONY: all

linux:
	@gox -arch="amd64" -os="linux" -ldflags=$(GO_LDFLAGS)

otherplats:
	@gox -arch="amd64" -os="windows darwin" -ldflags=$(GO_LDFLAGS)

build:
	@go build -buildmode=plugin -ldflags=$(GO_LDFLAGS) -o $(HANDLER).so

.PHONY: build

pack:
	@bash ci/shim/pack $(HANDLER) $(HANDLER).so $(PACKAGE).zip

.PHONY: pack

perm:
	@chown $(shell stat -c '%u:%g' .) $(HANDLER).so $(PACKAGE).zip

.PHONY: perm

clean:
	@rm -rf $(HANDLER).so $(PACKAGE).zip

.PHONY: clean
