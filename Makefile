
NAME=is-healthy
YQ=yq
OS   ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/')
LD_FLAGS=-ldflags "-w -s -X \"main.version=$(VERSION_TAG)\""
ifeq ($(VERSION),)
  VERSION_TAG=$(shell git describe --abbrev=0 --tags || echo latest)
else
  VERSION_TAG=$(VERSION)
endif


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: test
test:
	go test ./... -v

.PHONY: lint
lint: fmt
	golangci-lint run


.PHONY:
sync:
	git submodule update --init --recursive

update-submodules:
	git submodule update --remote --merge && git submodule sync


.PHONY: tidy
tidy: fmt
	go mod tidy

fmt:
	golines -m 120 -w pkg/
	golines -m 120 -w events/
	gofumpt -w .

.PHONY: compress
compress:
	test -e ./$(RELEASE_DIR)/$(NAME)_linux_amd64 && upx -5 ./$(RELEASE_DIR)/$(NAME)_linux_amd64 || true
	test -e ./$(RELEASE_DIR)/$(NAME)_linux_arm64 && upx -5 ./$(RELEASE_DIR)/$(NAME)_linux_arm64 || true

.PHONY: compress-build
compress-build:
	upx -5 ./$(RELEASE_DIR)/$(NAME) ./$(RELEASE_DIR)/$(NAME).test

.PHONY: linux
linux:
	GOOS=linux GOARCH=amd64 go build  -o ./$(RELEASE_DIR)/$(NAME)_linux_amd64 $(LD_FLAGS)  main.go
	GOOS=linux GOARCH=arm64 go build  -o ./$(RELEASE_DIR)/$(NAME)_linux_arm64 $(LD_FLAGS)  main.go

.PHONY: darwin
darwin:
	GOOS=darwin GOARCH=amd64 go build -o ./$(RELEASE_DIR)/$(NAME)_darwin_amd64 $(LD_FLAGS)  main.go
	GOOS=darwin GOARCH=arm64 go build -o ./$(RELEASE_DIR)/$(NAME)_darwin_arm64 $(LD_FLAGS)  main.go

.PHONY: windows
windows:
	GOOS=windows GOARCH=amd64 go build -o ./$(RELEASE_DIR)/$(NAME).exe $(LD_FLAGS)  main.go

.PHONY: binaries
binaries: linux darwin windows compress


.PHONY: build
build:
	GOOS=$(OS) GOARCH=$(ARCH) go build -o ./.bin/$(NAME) $(LD_FLAGS)  main.go


.PHONY: install
install:
	cp ./.bin/$(NAME) /usr/local/bin/
