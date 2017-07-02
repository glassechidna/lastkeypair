HANDLER ?= handler
PACKAGE ?= $(HANDLER)
GOPATH  ?= $(HOME)/go

all: build pack perm

.PHONY: all

build:
	@go build -buildmode=plugin -ldflags='-w -s' -o $(HANDLER).so

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
