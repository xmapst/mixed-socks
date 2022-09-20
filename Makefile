SHELL=/bin/bash
BINARY_NAME := mixed-socks
GIT_URL := https://github.com/xmapst/mixed-socks.git
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse HEAD)
VERSION := $(shell git describe --tags)
USER_NAME := $(shell git config user.name)
USER_EMAIL := $(shell git config user.email)
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S %Z")
LDFLAGS := "-w -s \
-X 'github/xmapst/$(BINARY_NAME).Version=$(VERSION)' \
-X 'github/xmapst/$(BINARY_NAME).GitUrl=$(GIT_URL)' \
-X 'github/xmapst/$(BINARY_NAME).GitBranch=$(GIT_BRANCH)' \
-X 'github/xmapst/$(BINARY_NAME).GitCommit=$(GIT_COMMIT)' \
-X 'github/xmapst/$(BINARY_NAME).BuildTime=$(BUILD_TIME)' \
-X 'github/xmapst/$(BINARY_NAME).UserName=$(USER_NAME)' \
-X 'github/xmapst/$(BINARY_NAME).UserEmail=$(USER_EMAIL)' \
"

all: windows linux

windows:
	@echo "build $(BINARY_NAME)_windows_amd64"
	@go mod tidy
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bin/$(BINARY_NAME)_windows_amd64.exe cmd/$(BINARY_NAME).go
	@strip --strip-unneeded bin/$(BINARY_NAME)_windows_amd64.exe
	@upx --lzma bin/$(BINARY_NAME)_windows_amd64.exe

linux:
	@echo "build $(BINARY_NAME)_linux_amd64"
	@go mod tidy
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bin/$(BINARY_NAME)_linux_amd64 cmd/$(BINARY_NAME).go
	@strip --strip-unneeded bin/$(BINARY_NAME)_linux_amd64
	@upx --lzma bin/$(BINARY_NAME)_linux_amd64