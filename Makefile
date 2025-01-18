.PHONY: clean test lint proxemb release-dry-run release

PACKAGE_NAME := github.com/eja/proxemb
GOLANG_CROSS_VERSION := v1.22.2
GOPATH ?= '$(HOME)/go'

all: lint proxemb

clean:
	@rm -f proxemb proxemb.exe

lint:
	@gofmt -w .

test:
	@go mod tidy
	@go mod verify
	@go vet ./...
	@go test -v ./test

proxemb:
	@go build -tags "fts5" -ldflags "-s -w" -o proxemb .
	@strip proxemb
