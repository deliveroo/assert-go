all: install lint test

install:
	@go install ./...

lint:
	@golangci-lint run ./...

setup:
	@go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	@dep ensure

test:
	@go test ./...


.PHONY: install lint setup test
