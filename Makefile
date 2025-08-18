.PHONY: lint test build

lint:
	golangci-lint run -v --fix ./...

fmt:
	gofmt -s -w ./...

test:
	go test ./...

build:
	go build -o feller main.go