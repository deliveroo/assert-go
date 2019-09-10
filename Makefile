all: install lint test

install:
	@go install ./...

lint:
	@golangci-lint run ./...

setup:
	@go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.18.0

test:
	@go test ./...


.PHONY: install lint setup test
