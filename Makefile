CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X github.com/trknhr/backlog-git-pr-diff-checker.revision=$(CURRENT_REVISION)"

deps:
	go get -u -d
	go mod tidy

.PHONY: build
build: deps
	go build -ldflags=$(BUILD_LDFLAGS) ./

.PHONY: install
install: deps
	go install -ldflags=$(BUILD_LDFLAGS) ./
