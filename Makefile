.PHONY: clean lint proxemb

PACKAGE_NAME := github.com/eja/proxemb
GOLANG_CROSS_VERSION := v1.22.2
GOPATH ?= '$(HOME)/go'

all: lint proxemb

clean:
	@rm -f proxemb proxemb.exe

lint:
	@gofmt -w .

proxemb:
	@go build -tags "fts5" -ldflags "-s -w" -o proxemb .
	@strip proxemb
